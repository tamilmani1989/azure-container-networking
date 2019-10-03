package dncclient

import (
	"bytes"
	"dnc/api"
	"dnc/datamodel"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DNC API fmt
const (
	apiVersion                 = "?api-version=2018-03-01"
	storageApi                 = "%s/azurestoragesettingsv3/blob/%s/container/%s" + apiVersion
	nodeApi                    = "%s/nodes/%s" + apiVersion
	networkApi                 = "%s/networks/%s" + apiVersion
	networkStatusApi           = "%s/networks/%s/status" + apiVersion
	reserveIpApi               = "%s/networks/%s/reservations/%s" + apiVersion
	reserveIpStatusApi         = "%s/networks/%s/reservations/%s/status" + apiVersion
	subnetApi                  = "%s/networks/%s/subnets" + apiVersion
	subnetStatusApi            = "%s/networks/%s/subnets/status" + apiVersion
	ncApi                      = "%s/networks/%s/networkcontainers/%s" + apiVersion
	ncStatusApi                = "%s/networks/%s/networkcontainers/%s/status" + apiVersion
	ncConfigurationApi         = "%s/networks/%s/networkcontainers/%s/configuration" + apiVersion
	reservationApi             = "%s/networks/%s/reservations/%s" + apiVersion
	promoteApi                 = "%s/promotenetworkcontainerstate/networks/%s/networkcontainer/%s" + apiVersion
	isWaitingForNodeCleanupApi = "%s/iswaitingfornodecleanup/networks/%s/networkcontainer/%s" + apiVersion
	skipNodeCleanupApi         = "%s/skipnodecleanupfornetworkcontainer/networks/%s/networkcontainer/%s" + apiVersion
)

const (
	jsonContentType string = "application/json"
	delete          string = "DELETE"
)

// OrchestratorContext - Struct to hold orchestrator context info.
type OrchestratorContext struct {
	PodName      string
	PodNamespace string
}

type DncClient struct {
	DncEP        string
	Blob         string
	Container    string
	PartitionKey string
	httpc        *http.Client
}

func NewDncClient(dncEP, blob, container, partitionKey string) *DncClient {
	return &DncClient{
		DncEP:        dncEP,
		Blob:         blob,
		Container:    container,
		PartitionKey: partitionKey,
		httpc:        &http.Client{Timeout: time.Second * 30}}
}

func (c *DncClient) InitAzStorage(accountName, key, partitionKey string) (err error) {
	if c.Blob == "" || c.Container == "" || c.PartitionKey == "" {
		return
	}

	var (
		body         bytes.Buffer
		respErr      api.JsonErr
		response     *http.Response
		storageURL   = fmt.Sprintf(storageApi, c.DncEP, c.Blob, c.Container)
		azStorageReq = api.AzureStorageRequest{
			AccountName:  accountName,
			Key:          key,
			PartitionKey: partitionKey}
	)

	json.NewEncoder(&body).Encode(azStorageReq)
	response, err = c.httpc.Post(storageURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] failed to initialize storage with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

/*
 * Add APIs
 */

func (c *DncClient) AddNode(nodeId, ipaddr, orchestratorType string) (err error) {
	var (
		body     bytes.Buffer
		respErr  api.JsonErr
		response *http.Response
		nodeURL  = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		nodeReq  = api.NodeInfoRequest{
			IPAddresses:      []string{ipaddr},
			LocationType:     "",
			OrchestratorType: orchestratorType}
	)

	json.NewEncoder(&body).Encode(nodeReq)
	response, err = c.httpc.Post(nodeURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] add node failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

func (c *DncClient) AddNetwork(networkId string) (err error) {
	var (
		body       bytes.Buffer
		respErr    api.JsonErr
		response   *http.Response
		networkURL = fmt.Sprintf(networkApi, c.DncEP, networkId)
		networkReq = api.NetworkInfoRequest{NetworkType: "AzureNet"}
	)

	json.NewEncoder(&body).Encode(networkReq)
	response, err = c.httpc.Post(networkURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] add network failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

func (c *DncClient) AddSubnet(subnetId, authenticationToken string) (err error) {
	var (
		body      bytes.Buffer
		respErr   api.JsonErr
		response  *http.Response
		subnetURL = fmt.Sprintf(subnetApi, c.DncEP, subnetId)
		subnetReq = api.SubnetRequest{AuthenticationToken: authenticationToken}
	)

	json.NewEncoder(&body).Encode(subnetReq)
	response, err = c.httpc.Post(subnetURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] add subnet failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

func (c *DncClient) AddNc(nodeId, networkId, ncId, reservationId, subnet,
	ipConstraint, nodeConstraint, ifaceId,
	containerType, podName, podNamespace string,
	host2NC, NC2Host bool) (err error) {
	var (
		body     bytes.Buffer
		respErr  api.JsonErr
		response *http.Response
		ncURL    = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		podInfo  = OrchestratorContext{
			PodName:      podName,
			PodNamespace: podNamespace}
		context, _      = json.Marshal(podInfo)
		associationInfo = api.AssociationInfo{
			NodeID:              nodeId,
			InterfaceID:         ifaceId,
			ContainerType:       containerType,
			OrchestratorContext: context}
	)

	// if reservation ID is empty, this is a V2 request
	if reservationId == "" {
		allocationReq := api.AllocationRequest{
			SubnetName:     subnet,
			IPConstraint:   ipConstraint,
			NodeConstraint: nodeConstraint}
		ncReq := api.NetworkContainerRequestV2{
			AllocationRequest:          allocationReq,
			AssociationInfo:            associationInfo,
			AllowHostToNCCommunication: host2NC,
			AllowNCToHostCommunication: NC2Host}
		json.NewEncoder(&body).Encode(ncReq)
	} else {
		ncReq := api.NetworkContainerRequestV1{
			ReservationID:              reservationId,
			AssociationInfo:            associationInfo,
			AllowHostToNCCommunication: host2NC,
			AllowNCToHostCommunication: NC2Host}
		json.NewEncoder(&body).Encode(ncReq)
	}

	response, err = c.httpc.Post(ncURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] add nc failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

func (c *DncClient) AddReservation(networkId, subnet, ipConstraint, nodeConstraint, reservationId string) (err error) {
	var (
		body           bytes.Buffer
		respErr        api.JsonErr
		response       *http.Response
		reservationURL = fmt.Sprintf(reservationApi, c.DncEP, networkId, reservationId)
		reservationReq = api.AllocationRequest{
			SubnetName:     subnet,
			IPConstraint:   ipConstraint,
			NodeConstraint: nodeConstraint}
	)

	json.NewEncoder(&body).Encode(reservationReq)
	response, err = c.httpc.Post(reservationURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] add reservation failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

/*
 * Get APIs
 */

func (c *DncClient) GetNode(nodeId string) (nodeCfg *api.NodeInfoResponse, err error) {
	var (
		response         *http.Response
		nodeURL          = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		nodeInfoResponse api.NodeInfoResponse
	)

	response, err = c.httpc.Get(nodeURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &nodeInfoResponse); err != nil {
				err = fmt.Errorf("[dnctl] get node failed with error %+v", err)
			} else {
				nodeCfg = &nodeInfoResponse
			}
		} else {
			err = fmt.Errorf("[dnctl] get node failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetNetworkConfiguration(networkId string) (networkCfg *api.NetworkConfigurationResponse, err error) {
	var (
		response                     *http.Response
		networkURL                   = fmt.Sprintf(networkApi, c.DncEP, networkId)
		networkConfigurationResponse api.NetworkConfigurationResponse
	)

	response, err = c.httpc.Get(networkURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &networkConfigurationResponse); err != nil {
				err = fmt.Errorf("[dnctl] get network failed with error %+v", err)
			} else {
				networkCfg = &networkConfigurationResponse
			}
		} else {
			err = fmt.Errorf("[dnctl] get network failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetNetworkStatus(networkId string) (requestStatus *api.RequestStatus, err error) {
	var (
		response         *http.Response
		networkStatusURL = fmt.Sprintf(networkStatusApi, c.DncEP, networkId)
		reqStatus        api.RequestStatus
	)

	response, err = c.httpc.Get(networkStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				err = fmt.Errorf("[dnctl] get network status failed with error %+v", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			err = fmt.Errorf("[dnctl] get network status failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetSubnetConfiguration(networkId, subnet string) (subnetCfg *api.SubnetConfigurationResponse, err error) {
	var (
		response                    *http.Response
		subnetURL                   = fmt.Sprintf(subnetApi, c.DncEP, networkId, subnet)
		subnetConfigurationResponse api.SubnetConfigurationResponse
	)

	response, err = c.httpc.Get(subnetURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &subnetConfigurationResponse); err != nil {
				err = fmt.Errorf("[dnctl] get subnet failed with error %+v", err)
			} else {
				subnetCfg = &subnetConfigurationResponse
			}
		} else {
			err = fmt.Errorf("[dnctl] get subnet failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetSubnetStatus(networkId, subnet string) (requestStatus *api.RequestStatus, err error) {
	var (
		response        *http.Response
		subnetStatusURL = fmt.Sprintf(subnetStatusApi, c.DncEP, networkId, subnet)
		reqStatus       api.RequestStatus
	)

	response, err = c.httpc.Get(subnetStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				err = fmt.Errorf("[dnctl] get subnet status failed with error %+v", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			err = fmt.Errorf("[dnctl] get subnet status failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetNC(networkId, ncId string) (nc *api.NetworkConfigurationResponse, err error) {
	var (
		response                *http.Response
		ncURL                   = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		ncConfigurationResponse api.NetworkConfigurationResponse
	)

	response, err = c.httpc.Get(ncURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &ncConfigurationResponse); err != nil {
				err = fmt.Errorf("[dnctl] get nc failed with error %+v", err)
			} else {
				nc = &ncConfigurationResponse
			}
		} else {
			err = fmt.Errorf("[dnctl] get nc failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetNCConfiguration(networkId, ncId string) (ncCfg *datamodel.NCConfiguration, err error) {
	var (
		response        *http.Response
		ncURL           = fmt.Sprintf(ncConfigurationApi, c.DncEP, networkId, ncId)
		ncConfiguration datamodel.NCConfiguration
	)

	response, err = c.httpc.Get(ncURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &ncConfiguration); err != nil {
				err = fmt.Errorf("[dnctl] get nc configuration failed with error %+v", err)
			} else {
				ncCfg = &ncConfiguration
			}
		} else {
			err = fmt.Errorf("[dnctl] get nc configuration failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetNCStatus(networkId, ncId string) (requestStatus *api.RequestStatus, err error) {
	var (
		response    *http.Response
		ncStatusURL = fmt.Sprintf(ncStatusApi, c.DncEP, networkId, ncId)
		reqStatus   api.RequestStatus
	)

	response, err = c.httpc.Get(ncStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				err = fmt.Errorf("[dnctl] get nc status failed with error %+v", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			err = fmt.Errorf("[dnctl] get nc status failed with error code: %d", response.StatusCode)
		}
	}
	return
}

func (c *DncClient) GetReservation(networkId, reservationId string) (reservationCfg *api.ReservationResponse, err error) {
	var (
		response            *http.Response
		reservationURL      = fmt.Sprintf(reservationApi, c.DncEP, networkId, reservationId)
		reservationResponse api.ReservationResponse
	)

	response, err = c.httpc.Get(reservationURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reservationResponse); err != nil {
				err = fmt.Errorf("[dnctl] get reservation failed with error %+v", err)
			} else {
				reservationCfg = &reservationResponse
			}
		} else {
			err = fmt.Errorf("[dnctl] get reservation failed with error code: %d", response.StatusCode)
		}
	}
	return
}

/*
 * Delete APIs
 */

func (c *DncClient) DeleteNode(nodeId string) (err error) {
	var (
		response *http.Response
		respErr  api.JsonErr
		nodeURL  = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		req, _   = http.NewRequest(delete, nodeURL, nil)
	)

	response, err = c.httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] delete node failed with error %+v", err)
			} else if respErr.HTTPStatus != http.StatusOK {
				err = fmt.Errorf("[dnctl] delete node failed with error %s", respErr.String())
			}
		}
	}
	return
}

func (c *DncClient) DeleteNetwork(networkId string) (err error) {
	var (
		response   *http.Response
		respErr    api.JsonErr
		networkURL = fmt.Sprintf(networkApi, c.DncEP, networkId)
		req, _     = http.NewRequest(delete, networkURL, nil)
	)

	response, err = c.httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] delete network failed with error %+v", err)
			} else if respErr.HTTPStatus != http.StatusOK {
				err = fmt.Errorf("[dnctl] delete network failed with error %s", respErr.String())
			}
		}
	}
	return
}

func (c *DncClient) DeleteSubnet(networkId, subnet string) (err error) {
	var (
		response  *http.Response
		respErr   api.JsonErr
		subnetURL = fmt.Sprintf(subnetApi, c.DncEP, networkId, subnet)
		req, _    = http.NewRequest(delete, subnetURL, nil)
	)

	response, err = c.httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] delete subnet failed with error %+v", err)
			} else if respErr.HTTPStatus != http.StatusOK {
				err = fmt.Errorf("[dnctl] delete subnet failed with error %s", respErr.String())
			}
		}
	}
	return
}

func (c *DncClient) DeleteNC(networkId, ncId string) (err error) {
	var (
		response *http.Response
		respErr  api.JsonErr
		ncURL    = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		req, _   = http.NewRequest(delete, ncURL, nil)
	)

	response, err = c.httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] delete NC failed with error %+v", err)
			} else if respErr.HTTPStatus != http.StatusOK {
				err = fmt.Errorf("[dnctl] delete NC failed with error %s", respErr.String())
			}
		}
	}
	return
}

/*
 * Debug APIs
 */

func (c *DncClient) PromoteNCState(networkId, ncId string) (err error) {
	var (
		body       bytes.Buffer
		respErr    api.JsonErr
		response   *http.Response
		promoteURL = fmt.Sprintf(promoteApi, c.DncEP, networkId, ncId)
	)

	response, err = c.httpc.Post(promoteURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] promote NC failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}
	return
}

func (c *DncClient) IsWaitingForNodeCleanup(networkId, ncId string) (isWaitingForNodeCleanup bool) {
	var (
		body                            bytes.Buffer
		err                             error
		response                        *http.Response
		isWaitingForNodeCleanupResponse api.IsWaitingForNodeCleanupResponse
		isWaitingForNodeCleanupURL      = fmt.Sprintf(isWaitingForNodeCleanupApi, c.DncEP, networkId, ncId)
	)

	response, err = c.httpc.Post(isWaitingForNodeCleanupURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &isWaitingForNodeCleanupResponse); err != nil {
				err = fmt.Errorf("[dnctl] failed to check if NC is in node cleanup state with err %+v", err)
			} else {
				isWaitingForNodeCleanup = isWaitingForNodeCleanupResponse.IsWaitingForNodeCleanup
			}
		}
	}
	return
}

func (c *DncClient) SkipNodeCleanupForNC(networkId, ncId string) (err error) {
	var (
		body               bytes.Buffer
		response           *http.Response
		respErr            api.JsonErr
		skipNodeCleanupURL = fmt.Sprintf(skipNodeCleanupApi, c.DncEP, networkId, ncId)
	)

	response, err = c.httpc.Post(skipNodeCleanupURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("[dnctl] failed to skip node cleanup for NC with err %+v", err)
			}
		}
	}
	return
}
