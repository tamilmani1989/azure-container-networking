package aznm

import (
	"context"
	"crypto/tls"
	"dnc/logger/loggershim"
	"dnc/microsoft/azure/aznm/contracts"
	"dnc/microsoft/azure/aznm/contracts/containersprotocol"
	"fmt"
	localtls "github.com/Azure/azure-container-networking/server/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
	"time"
)

var (
	apiTimeout        = 5 * time.Minute // TODO: figure out appropriate timeout from aznm team
	connectionTimeout = 2 * time.Minute // TODO: figure out appropriate timeout from aznm team
)

// Client to connect to AZNM over grpc protocol. It encapsulates
// grpc connection. Note: Client does caching of connections for a
// given endpoint. If connection exists for an endpoint, it won't be recreated.
// Same connection can be used to multiplex RPC calls and is safe to
// be used across go routines. If grpc connection is broken,
// it itself tries to reconnect too. Only limitation could be if a
// single connection is used across log of go routines and it reaches stream limit.
// We will need to load test aznm client to see if single instance (per endpoint) works
// or we need to use a connection pool. https://github.com/grpc/grpc-go/issues/682
type AznmGrpcClient struct {
	lock     sync.Mutex
	clients  map[string]containersprotocol.NetworkContainerServiceClient
	logger   loggershim.LoggerInterface
	certPath string
}

// endpoint - fqdn:port or ipaddress:port
//     example: uscentraleuap-az01.aznm.core.windows.net:444
//     example: 13.104.208.31:444
// activityId - random unique id to log requests on server side for API call.
type AznmRequest struct {
	Endpoint   string
	ActivityId string
}

// certPath - cert needed to authenticate with aznm
// logger - logger that implements loggerInterface
func NewAznmGrpcClient(certPath string, logger loggershim.LoggerInterface) *AznmGrpcClient {
	return &AznmGrpcClient{
		certPath: certPath,
		logger:   logger,
		clients: make(map[string]containersprotocol.NetworkContainerServiceClient),
	}
}

// reqContext - context to call into AZNM
// logicalId - logical container id. Should be Guid.
// physicalId - physicalId container id. Should be Guid.
// bladeId - Azure blade or node id. Should be Guid.
func (c *AznmGrpcClient) UpdateContainer(
	ctx context.Context,
	aznmRequest AznmRequest,
	logicalId, physicalId, bladeId string) error {

	bladeUuid, err := convertGuidStrToProtobufUuid(bladeId)
	if err != nil {
		return fmt.Errorf("AznmGrpcClient(UpdateContainer): Failed to convert bladeId %s to uuid: %s",
			bladeId, err)
	}

	logicalUuid, err := convertGuidStrToProtobufUuid(logicalId)
	if err != nil {
		return fmt.Errorf("AznmGrpcClient(UpdateContainer): Failed to convert logicalId %s to uuid: %s",
			logicalId, err)
	}

	physicalUuid, err := convertGuidStrToProtobufUuid(physicalId)
	if err != nil {
		return fmt.Errorf("AznmGrpcClient(UpdateContainer): Failed to convert physicalId %s to uuid: %s",
			physicalId, err)
	}

	// TODO : pass activity id via parameters "accf7771-bece-47d1-a3ee-13b9d129dd71"
	req := &containersprotocol.UpdateContainerRequest{
		Context: &contracts.RequestContext{ActivityIds: []string{aznmRequest.ActivityId}},
		Container: &containersprotocol.Container{
			Type: &containersprotocol.Container_BareMetalContainer{
				BareMetalContainer: &containersprotocol.BareMetalContainer{
					Id:                 physicalUuid,
					LogicalContainerId: logicalUuid,
					Version:            1,
					BladeId:            bladeUuid,
					IsActivated:        true,
				},
			},
		},
	}

	client, err := c.getAznmClient(ctx, aznmRequest)
	if err != nil {
		return err
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	// response struct from server is empty, so can be ignored for now
	_, err = client.UpdateContainer(localCtx, req)
	return err
}

// reqContext - context to call into AZNM
// activityId - random unique id to log requests on server side for API call.
// logicalId - logical container id. Should be Guid.
// physicalId - physicalId container id. Should be Guid.
func (c *AznmGrpcClient) DeleteContainer(
	ctx context.Context,
	aznmRequest AznmRequest,
	logicalId, physicalId string) error {

	logicalUuid, err := convertGuidStrToProtobufUuid(logicalId)
	if err != nil {
		return fmt.Errorf("AznmGrpcClient(DeleteContainer): Failed to convert logicalId %s to uuid: %s",
			logicalId, err)
	}

	physicalUuid, err := convertGuidStrToProtobufUuid(physicalId)
	if err != nil {
		return fmt.Errorf("AznmGrpcClient(DeleteContainer): Failed to convert physicalId %s to uuid: %s",
			physicalId, err)
	}

	// TODO : pass activity id via parameters "accf7771-bece-47d1-a3ee-13b9d129dd71"
	req := &containersprotocol.DeleteContainerRequest{
		Context: &contracts.RequestContext{ActivityIds: []string{aznmRequest.ActivityId}},
		ContainerIdentifier: &contracts.ContainerIdentifier{
			LogicalContainerId: logicalUuid,
			ContainerId:        physicalUuid,
		},
	}

	client, err := c.getAznmClient(ctx, aznmRequest)
	if err != nil {
		return err
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	_, err = client.DeleteContainer(localCtx, req)
	return err
}

func (c *AznmGrpcClient) getAznmClient(
	ctx context.Context,
	reqContext AznmRequest) (containersprotocol.NetworkContainerServiceClient, error) {

	c.lock.Lock()
	defer c.lock.Unlock()

	var client containersprotocol.NetworkContainerServiceClient
	var err error
	var ok bool

	client, ok = c.clients[reqContext.Endpoint]
	if !ok {
		client, err = c.newAznmGrpcClient(ctx, reqContext)
		if err != nil {
			return nil, err
		}

		c.clients[reqContext.Endpoint] = client
		c.logger.Debugf("client created and stored in map for endpoint %s", reqContext.Endpoint)
	}

	return client, nil
}

func (c *AznmGrpcClient) newAznmGrpcClient(
	ctx context.Context,
	aznmReq AznmRequest) (containersprotocol.NetworkContainerServiceClient, error) {

	tlsConfig, err := c.getTLSConfig()
	if err != nil {
		return nil, err
	}

	localCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		localCtx, aznmReq.Endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("grpc connect failed for endpoint %s with error: %s", aznmReq.Endpoint, err)
	}

	c.logger.Debugf("grpc connection succeeded to endpoint :%s", aznmReq.Endpoint)

	return containersprotocol.NewNetworkContainerServiceClient(conn), nil
}

func (c *AznmGrpcClient) getTLSConfig() (*tls.Config, error) {
	tlsSettings := localtls.TlsSettings{
		TLSCertificatePath: c.certPath,
	}
	tlsCertRetriever, err := localtls.GetTlsCertificateRetriever(tlsSettings)
	if err != nil {
		return nil, fmt.Errorf("AznmGrpcClient: Failed to get certificate retriever: %s", err)
	}

	leafCertificate, err := tlsCertRetriever.GetCertificate()
	if err != nil {
		return nil, fmt.Errorf("AznmGrpcClient: Failed to get certificate: %s", err)
	}

	if leafCertificate == nil {
		return nil, fmt.Errorf("AznmGrpcClient: Certificate retrival returned empty: %s", err)
	}

	c.logger.Debugf("AznmGrpcClient: Retrieved leaf certificate successfully")

	privateKey, err := tlsCertRetriever.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("AznmGrpcClient: Failed to get certificate private key: %s", err)
	}

	c.logger.Debugf("AznmGrpcClient: Retrieved private key succesfully")

	tlsCert := tls.Certificate{
		Certificate: [][]byte{leafCertificate.Raw},
		PrivateKey:  privateKey,
		Leaf:        leafCertificate,
	}

	// InsecureSkipVerify : false will ensure client doesn't verify server side
	// certificate. If AZNM server certificate needs to be verified, then we will
	// need to set RootCAs of server side certificate in this tls config
	tlsConfig := &tls.Config{
		MaxVersion: tls.VersionTLS12,
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			tlsCert,
		},
		InsecureSkipVerify: true,
	}

	return tlsConfig, nil
}