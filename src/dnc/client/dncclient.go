package client

import (
	"bytes"
	"dnc/api"
	"dnc/datamodel"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	reservationApi             = "%s/networks/%s/reservations/%s" + apiVersion
	reservationStatusApi       = "%s/networks/%s/reservations/%s/status" + apiVersion
	subnetApi                  = "%s/networks/%s/subnets/%s" + apiVersion
	subnetStatusApi            = "%s/networks/%s/subnets/%s/status" + apiVersion
	ncApi                      = "%s/networks/%s/networkcontainer/%s" + apiVersion
	ncStatusApi                = "%s/networks/%s/networkcontainers/%s/status" + apiVersion
	ncConfigurationApi         = "%s/networks/%s/networkcontainer/%s/configuration" + apiVersion
	ncCleanupApi               = "%s/networks/%s/networkcontainercleanupstatus/%s" + apiVersion
	promoteApi                 = "%s/promotenetworkcontainerstate/networks/%s/networkcontainer/%s" + apiVersion
	isWaitingForNodeCleanupApi = "%s/iswaitingfornodecleanup/networks/%s/networkcontainer/%s" + apiVersion
	skipNodeCleanupApi         = "%s/skipnodecleanupfornetworkcontainer/networks/%s/networkcontainer/%s" + apiVersion
)

const (
	jsonContentType string = "application/json"
	delete          string = "DELETE"
)

var (
	// our default http client with 30s timeout
	httpc = &http.Client{Timeout: time.Second * 30}
)

// OrchestratorContext - Struct to hold orchestrator context info
type OrchestratorContext struct {
	PodName      string
	PodNamespace string
}

// DncClient - Struct to hold DNC cluster specific info
type DncClient struct {
	DncEP string
}

func NewDncClient(dncEP string) *DncClient {
	return &DncClient{
		DncEP: dncEP}
}

func (c *DncClient) InitAzStorage(accountName, key, partitionKey, blob, container string) (respErr *api.JsonErr) {
	if blob == "" || container == "" {
		return
	}

	var (
		body         bytes.Buffer
		response     *http.Response
		err          error
		storageURL   = fmt.Sprintf(storageApi, c.DncEP, blob, container)
		azStorageReq = api.AzureStorageRequest{
			AccountName:  accountName,
			Key:          key,
			PartitionKey: partitionKey}
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	json.NewEncoder(&body).Encode(azStorageReq)
	response, err = httpc.Post(storageURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, AzStorage, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

/*
 * Add APIs
 */

func (c *DncClient) AddNode(nodeId, ipaddr, orchestratorType, infraVnet string) (respErr *api.JsonErr) {
	var (
		body     bytes.Buffer
		response *http.Response
		err      error
		nodeURL  = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		nodeReq  = api.NodeInfoRequest{
			IPAddresses:           []string{ipaddr},
			LocationType:          "",
			OrchestratorType:      orchestratorType,
			InfrastructureNetwork: infraVnet}
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	json.NewEncoder(&body).Encode(nodeReq)
	response, err = httpc.Post(nodeURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, respErr); err != nil {
				failedToDecodeResponse(respErr, Register, Node, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) AddNetwork(networkId string) (respErr *api.JsonErr) {
	var (
		body       bytes.Buffer
		response   *http.Response
		err        error
		networkURL = fmt.Sprintf(networkApi, c.DncEP, networkId)
		networkReq = api.NetworkInfoRequest{NetworkType: "AzureNet"}
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	json.NewEncoder(&body).Encode(networkReq)
	response, err = httpc.Post(networkURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, Network, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) AddSubnet(networkId, subnetId, authenticationToken string) (respErr *api.JsonErr) {
	var (
		body      bytes.Buffer
		response  *http.Response
		err       error
		subnetURL = fmt.Sprintf(subnetApi, c.DncEP, networkId, subnetId)
		subnetReq = api.SubnetRequest{AuthenticationToken: authenticationToken}
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	json.NewEncoder(&body).Encode(subnetReq)
	response, err = httpc.Post(subnetURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, Subnet, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) AddNc(nodeId, networkId, ncId, reservationId, subnet,
	ipConstraint, nodeConstraint, ifaceId,
	containerType, podName, podNamespace string,
	host2NC, NC2Host bool, secondaryIPCount int) (respErr *api.JsonErr) {
	var (
		body     bytes.Buffer
		response *http.Response
		err      error
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
			SubnetName:       subnet,
			IPConstraint:     ipConstraint,
			NodeConstraint:   nodeConstraint,
			SecondaryIPCount: secondaryIPCount}
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

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Post(ncURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, NetworkContainer, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) UpsertNcV2(networkId, ncId string, req api.NetworkContainerRequestV2) *api.JsonErr {
	var (
		body     bytes.Buffer
		response *http.Response
		err      error
		ncURL                 = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		respErr  *api.JsonErr = &api.JsonErr{}
	)

	err = json.NewEncoder(&body).Encode(req)
	if err != nil {
		respErr.Text = err.Error()
		return respErr
	}

	httpReq, err := http.NewRequest(http.MethodPut, ncURL, &body)
	if err != nil {
		respErr.Text = err.Error()
		return respErr
	}
	httpReq.Header.Add("Content-Type", jsonContentType)

	response, err = httpc.Do(httpReq)
	respErr.HTTPStatus = http.StatusOK
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, NetworkContainer, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}

	return respErr
}

// AddReservation will call DNC client to create a IP reservation.
func (c *DncClient) AddReservation(networkId, subnet, ipConstraint, nodeConstraint, reservationId, requestType string) (respErr *api.JsonErr) {
	var (
		body           bytes.Buffer
		response       *http.Response
		err            error
		reservationURL = fmt.Sprintf(reservationApi, c.DncEP, networkId, reservationId)
		reservationReq = api.AllocationRequest{
			SubnetName:     subnet,
			IPConstraint:   ipConstraint,
			NodeConstraint: nodeConstraint,
			RequestType:    requestType}
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	json.NewEncoder(&body).Encode(reservationReq)
	response, err = httpc.Post(reservationURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Register, Reservation, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

/*
 * Get APIs
 */

func (c *DncClient) GetNode(nodeId string) (nodeCfg *api.NodeInfoResponse, respErr *api.JsonErr) {
	var (
		response         *http.Response
		err              error
		nodeURL          = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		nodeInfoResponse api.NodeInfoResponse
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(nodeURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &nodeInfoResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Node, err)
			} else {
				nodeCfg = &nodeInfoResponse
			}
		} else {
			failedWithStatus(respErr, Get, Node, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetNetworkConfiguration(networkId string) (networkCfg *api.NetworkConfigurationResponse, respErr *api.JsonErr) {
	var (
		response                     *http.Response
		err                          error
		networkURL                   = fmt.Sprintf(networkApi, c.DncEP, networkId)
		networkConfigurationResponse api.NetworkConfigurationResponse
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(networkURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &networkConfigurationResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Network, err)
			} else {
				networkCfg = &networkConfigurationResponse
			}
		} else {
			failedWithStatus(respErr, Get, Network, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetNetworks() (*api.NetworksResponse, *api.JsonErr) {
	var (
		response         *http.Response
		err              error
		networksURL      = fmt.Sprintf(networkApi, c.DncEP, "")
		networksResponse api.NetworksResponse
		respErr          *api.JsonErr
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(networksURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &networksResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Network, err)
			} else {
				return &networksResponse, respErr
			}
		} else {
			failedWithStatus(respErr, Get, Network, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}

	return &networksResponse, respErr
}

func (c *DncClient) GetNetworkStatus(networkId string) (requestStatus *api.RequestStatus, respErr *api.JsonErr) {
	var (
		response         *http.Response
		err              error
		networkStatusURL = fmt.Sprintf(networkStatusApi, c.DncEP, networkId)
		reqStatus        api.RequestStatus
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(networkStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				failedToDecodeResponse(respErr, Get, Network+" status", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			failedWithStatus(respErr, Get, Network+" status", response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetSubnetConfiguration(networkId, subnet string) (subnetCfg *api.SubnetConfigurationResponse, respErr *api.JsonErr) {
	var (
		response                    *http.Response
		err                         error
		subnetURL                   = fmt.Sprintf(subnetApi, c.DncEP, networkId, subnet)
		subnetConfigurationResponse api.SubnetConfigurationResponse
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(subnetURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &subnetConfigurationResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Subnet, err)
			} else {
				subnetCfg = &subnetConfigurationResponse
			}
		} else {
			failedWithStatus(respErr, Get, Subnet, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetSubnetStatus(networkId, subnet string) (requestStatus *api.RequestStatus, respErr *api.JsonErr) {
	var (
		response        *http.Response
		err             error
		subnetStatusURL = fmt.Sprintf(subnetStatusApi, c.DncEP, networkId, subnet)
		reqStatus       api.RequestStatus
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(subnetStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				failedToDecodeResponse(respErr, Get, Subnet+" status", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			failedWithStatus(respErr, Get, Subnet+" status", response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetNC(networkId, ncId string) (nc *api.NetworkContainerResponse, respErr *api.JsonErr) {
	var (
		response                *http.Response
		err                     error
		ncURL                   = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		ncConfigurationResponse api.NetworkContainerResponse
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(ncURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &ncConfigurationResponse); err != nil {
				failedToDecodeResponse(respErr, Get, NetworkContainer, err)
			} else {
				nc = &ncConfigurationResponse
			}
		} else {
			failedWithStatus(respErr, Get, NetworkContainer, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetNCConfiguration(networkId, ncId string) (ncCfg *datamodel.NCConfiguration, respErr *api.JsonErr) {
	var (
		response        *http.Response
		err             error
		ncURL           = fmt.Sprintf(ncConfigurationApi, c.DncEP, networkId, ncId)
		ncConfiguration datamodel.NCConfiguration
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(ncURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &ncConfiguration); err != nil {
				failedToDecodeResponse(respErr, Get, NetworkContainer+" configuration", err)
			} else {
				ncCfg = &ncConfiguration
			}
		} else {
			failedWithStatus(respErr, Get, NetworkContainer+" configuration", response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

// GetNetworkContainerCleanupStatus returns the cleanup status of the nc
func (c *DncClient) GetNetworkContainerCleanupStatus(networkId, ncId string) (api.NetworkContainerCleanupResponse, *api.JsonErr) {
	ncCleanupURL := fmt.Sprintf(ncCleanupApi, c.DncEP, networkId, ncId)

	respErr := &api.JsonErr{HTTPStatus: http.StatusOK}
	var cleanupResponse api.NetworkContainerCleanupResponse

	response, err := httpc.Get(ncCleanupURL)
	if err == nil {
		defer response.Body.Close()
		respErr.HTTPStatus = response.StatusCode

		switch statusCode := response.StatusCode; statusCode {
		case http.StatusOK:
			if err := utils.DecodeResponse(response, &cleanupResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Subnet+" cleanup status", err)
			}

		default:
			failedWithStatus(respErr, Get, Subnet+" cleanup status", statusCode)
		}
	} else {
		// Some http error occurred
		reportClientErr(respErr, err)
	}

	return cleanupResponse, respErr
}

func (c *DncClient) GetNCStatus(networkId, ncId string) (requestStatus *api.RequestStatus, respErr *api.JsonErr) {
	var (
		response    *http.Response
		err         error
		ncStatusURL = fmt.Sprintf(ncStatusApi, c.DncEP, networkId, ncId)
		reqStatus   api.RequestStatus
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(ncStatusURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reqStatus); err != nil {
				failedToDecodeResponse(respErr, Get, NetworkContainer+" status", err)
			} else {
				requestStatus = &reqStatus
			}
		} else {
			bs, _ := ioutil.ReadAll(response.Body)
			failedWithStatus(respErr, Get, NetworkContainer+" status", response.StatusCode)
			respErr.Text = fmt.Sprintf("%s: %s", respErr.Text, string(bs))
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetReservation(networkId, reservationId string) (reservationCfg *api.ReservationResponse, respErr *api.JsonErr) {
	var (
		response            *http.Response
		err                 error
		reservationURL      = fmt.Sprintf(reservationApi, c.DncEP, networkId, reservationId)
		reservationResponse api.ReservationResponse
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(reservationURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reservationResponse); err != nil {
				failedToDecodeResponse(respErr, Get, Reservation, err)
			} else {
				reservationCfg = &reservationResponse
			}
		} else {
			failedWithStatus(respErr, Get, Reservation, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) GetReservationStatus(networkId, reservationId string) (reservationStatus *api.RequestStatus, respErr *api.JsonErr) {
	var (
		response              *http.Response
		err                   error
		reservationURL        = fmt.Sprintf(reservationStatusApi, c.DncEP, networkId, reservationId)
		reservationStatusResp api.RequestStatus
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Get(reservationURL)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			if err = utils.DecodeResponse(response, &reservationStatusResp); err != nil {
				failedToDecodeResponse(respErr, Get, Reservation, err)
			} else {
				reservationStatus = &reservationStatusResp
			}
		} else {
			failedWithStatus(respErr, Get, Reservation, response.StatusCode)
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

/*
 * Delete APIs
 */

func (c *DncClient) DeleteNode(nodeId string) (respErr *api.JsonErr) {
	var (
		response *http.Response
		err      error
		nodeURL  = fmt.Sprintf(nodeApi, c.DncEP, nodeId)
		req, _   = http.NewRequest(delete, nodeURL, nil)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Delete, Node, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) DeleteNetwork(networkId string) (respErr *api.JsonErr) {
	var (
		response   *http.Response
		err        error
		networkURL = fmt.Sprintf(networkApi, c.DncEP, networkId)
		req, _     = http.NewRequest(delete, networkURL, nil)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Delete, Network, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) DeleteSubnet(networkId, subnet string) (respErr *api.JsonErr) {
	var (
		response  *http.Response
		err       error
		subnetURL = fmt.Sprintf(subnetApi, c.DncEP, networkId, subnet)
		req, _    = http.NewRequest(delete, subnetURL, nil)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Delete, Subnet, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) DeleteNC(networkId, ncId string) (respErr *api.JsonErr) {
	var (
		response *http.Response
		err      error
		ncURL    = fmt.Sprintf(ncApi, c.DncEP, networkId, ncId)
		req, _   = http.NewRequest(delete, ncURL, nil)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Delete, NetworkContainer, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

func (c *DncClient) DeleteReservation(networkId, reservationId string) (respErr *api.JsonErr) {
	var (
		response       *http.Response
		err            error
		reservationURL = fmt.Sprintf(reservationApi, c.DncEP, networkId, reservationId)
		req, _         = http.NewRequest(delete, reservationURL, nil)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Do(req)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Delete, Reservation, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
	}
	return
}

/*
 * Debug APIs
 */

func (c *DncClient) PromoteNCState(networkId, ncId string) (respErr *api.JsonErr) {
	var (
		body       bytes.Buffer
		err        error
		response   *http.Response
		promoteURL = fmt.Sprintf(promoteApi, c.DncEP, networkId, ncId)
	)

	respErr = &api.JsonErr{HTTPStatus: http.StatusOK}
	response, err = httpc.Post(promoteURL, jsonContentType, &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = utils.DecodeResponse(response, &respErr); err != nil {
				failedToDecodeResponse(respErr, Promote, NetworkContainer, err)
			}
		}
	} else {
		reportClientErr(respErr, err)
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

	response, err = httpc.Post(isWaitingForNodeCleanupURL, jsonContentType, &body)
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

	response, err = httpc.Post(skipNodeCleanupURL, jsonContentType, &body)
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

func failedToDecodeResponse(respErr *api.JsonErr, action, resource string, err error) {
	respErr.Code = api.DecodeResponseFailed
	respErr.Text = fmt.Sprintf("[dnctl] failed to decode %s %s response %+v", action, resource, err)
	respErr.HTTPStatus = http.StatusInternalServerError
}

func failedWithStatus(respErr *api.JsonErr, action, resource string, statusCode int) {
	respErr.HTTPStatus = statusCode
	respErr.Text = fmt.Sprintf("[dnctl] failed to %s %s with status %d", action, resource, statusCode)
}

func reportClientErr(respErr *api.JsonErr, err error) {
	respErr.HTTPStatus = http.StatusAccepted
	respErr.Text = err.Error()
}
