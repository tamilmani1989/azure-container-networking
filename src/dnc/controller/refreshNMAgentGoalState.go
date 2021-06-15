package controller

import (
	"bytes"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (c *Controller) getNCSessionInfo(request *datamodel.NCConfiguration) map[string]string {
	sessionInfo := make(map[string]string)
	if len(request.NetworkID) != 0 {
		sessionInfo[datamodel.NetworkRepositoryName] = request.NetworkID
	}
	return sessionInfo
}

// ReJoinAllNetworks - Rejoins all the networks
func (c *Controller) ReJoinAllNetworks(networksMap map[string]bool) {
	iteration := 0
	for ok := true; ok; ok = len(networksMap) > 0 {
		iteration++
		index := 0
		failedNetworks := make(map[string]bool)
		for networkID := range networksMap {
			_, err, _ := c.postRequestToAgentForNetworkJoin(networkID)

			time.Sleep(1 * time.Second)

			if err != nil {
				loggershim.LS.Errorf("%s", err.String())
				failedNetworks[networkID] = true
			}
			index++
		}

		networksMap = failedNetworks

		if iteration > 10 {
			loggershim.LS.Debugf("Exhausted retries. Current retry count:%d. Failed network count:%d.", iteration, len(failedNetworks))
			break
		}

	}
}

func (c *Controller) RefreshNetworkContainersNMAgentGoalState() {
	ncsMap, err := datamodel.NcRepo.GetAllNetworkContainers()
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve network containers from repository due to error %s", err.Error())
		return
	}

	networksMap := make(map[string]bool)
	for _, nc := range ncsMap {
		networksMap[nc.Config.NetworkID] = true
	}

	c.ReJoinAllNetworks(networksMap)

	iteration := 0
	for ok := true; ok; ok = len(ncsMap) > 0 {
		iteration++
		index := 0
		failedNCsMap := make(map[string]datamodel.NetworkContainerEntity)
		for ncKey, nc := range ncsMap {
			sessionInfo := c.getNCSessionInfo(&nc.Config)
			_, err := c.refreshNetworkContainerNMAgentGoalState(ncKey, sessionInfo)

			time.Sleep(1 * time.Second)

			if err != nil {
				loggershim.LS.Errorf("Failed to refresh NMAgent goal state for NC %s with error:%+v", ncKey, err)
				failedNCsMap[ncKey] = nc
			}
			index++
		}

		ncsMap = failedNCsMap

		if iteration > 10 {
			loggershim.LS.Debugf("Exhausted retries. Current retry count:%d. Failed NCs count:%d.", iteration, len(failedNCsMap))
			break
		}

	}
}

func (c *Controller) refreshNetworkContainerNMAgentGoalState(ncKey string,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	sessionInfoWrite := map[string]string{datamodel.NCRepositoryName: datamodel.GetNCIdFromKey(ncKey)}
	c.SessionManager.BeginWriteSession(sessionInfoWrite)
	defer c.SessionManager.EndWriteSession(sessionInfoWrite)

	repoNCData, err := datamodel.NcRepo.TryQueryNetworkContainerByKey(ncKey)
	if err != nil {
		return false, err
	}

	if repoNCData.RowKey == "" {
		return true, nil
	}

	if repoNCData.Config.MarkedForDelete {
		return true, nil
	}

	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: repoNCData.Config.ReservationID,
		datamodel.NetworkRepositoryName:     repoNCData.Config.NetworkID,
		datamodel.NodeRepositoryName:        repoNCData.Config.GetAssociationInfo().NodeID,
	}
	c.SessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, _, ncGoalStateErr := datamodel.GetNCGoalState(&repoNCData.Config)
	c.SessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		return false, ncGoalStateErr
	}

	var body bytes.Buffer
	var ncRequest nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	ncRequest.IPV4Addresses = make([]string, 1)
	ncRequest.IPV4Addresses[0] = strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	ncRequest.Policies = make([]string, 0)
	ncRequest.SubnetName = ncGoalState.BaseSubnetName
	ncRequest.Version = strconv.FormatUint(ncGoalState.VnetVersion, 10)
	ncRequest.VirtualNetworkID = ncGoalState.NetworkID
	ncRequest.VlanID = ncGoalState.VlanID
	ncRequest.GreKey = int(ncGoalState.GreKey)

	json.NewEncoder(&body).Encode(ncRequest)

	createNCURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.PutNetworkContainerURLFmt,
		config.HostAgentSettings.WireServerIP,
		ncGoalState.NodeInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)

	resp, respErr := c.httpClient.Post(createNCURL, "application/json", &body)
	if respErr != nil {
		return false, api.InternalServerErrorf(api.PublishingFailed,
			"could not publish NC goal state for NC: %s to NMAgent. error: %s. Marking as retryable error.",
			repoNCData.Config.NetworkContainerID, respErr.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, api.InternalServerErrorf(api.PublishingFailed,
			"could not publish NC goal state for NC: %s to NMAgent. error code: %d. Marking as retryable error.",
			repoNCData.Config.NetworkContainerID, resp.StatusCode)
	}

	var createNetworkContainerResponse nmagentcontracts.CreateNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &createNetworkContainerResponse)
	if decodeRespErr != nil {
		return false, api.InternalServerErrorf(api.DecodeResponseFailed,
			"could not decode putNC response from NMAgent for NC: %s. error: %s", repoNCData.Config.NetworkContainerID, decodeRespErr.Error())
	}

	loggershim.LS.Debugf("[NMAgent][Response] refresh %s\n", createNetworkContainerResponse.String())
	putNCResponseCode, parseErr := strconv.ParseInt(createNetworkContainerResponse.ResponseCode, 10, 32)
	if parseErr != nil {
		return false, api.InternalServerErrorf(api.ParseResponseCodeFailed,
			"could not convert response code: %s to int when parsing putNC response for NC: %s. error: %s",
			createNetworkContainerResponse.ResponseCode, repoNCData.Config.NetworkContainerID, parseErr.Error())
	}

	if putNCResponseCode != http.StatusOK {
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       fmt.Sprintf("Failed to publish NC with status code %v.", putNCResponseCode),
			HTTPStatus: int(putNCResponseCode)}
	}

	return true, nil
}

// Post request to host agent to express the intent for joining a network.
// Returns IsRetirbale, optional error if any, isThrottled
func (c *Controller) postRequestToAgentForNetworkJoin(networkID string) (bool, *api.JsonErr, bool) {
	joinURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	loggershim.LS.Debugf("[NMAgent] joining network [%s]", networkID)

	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	joinResponse, joinRespErr := c.httpClient.Post(joinURL, "application/json", &body)
	if joinRespErr != nil {
		msg := fmt.Sprintf(
			"Join for network [%s] failed with reponse error [%s]. Marking this error as retriable",
			networkID,
			joinRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.JoinDenied,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, false
	}

	defer joinResponse.Body.Close()

	if joinResponse.StatusCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"Join for network [%s] is pending. Marking this error as retriable",
			networkID)
		return true, &api.JsonErr{
			Code:       api.JoinPending,
			Text:       msg,
			HTTPStatus: http.StatusProcessing}, false
	}

	if joinResponse.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network [%s] is unauthorized. Marking this error as non retriable.", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}, false
	}

	if joinResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Join for network [%s] failed with status [%d]. Marking this error as retriable.",
			networkID,
			joinResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, joinResponse.StatusCode == http.StatusForbidden
	}

	return false, nil, false
}
