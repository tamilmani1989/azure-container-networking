package mock

import (
	"context"
	"dnc/microsoft/azure/aznm/contracts/containersprotocol"
	"fmt"
	"github.com/Azure/azure-container-networking/log"
	"google.golang.org/grpc"
	"net"
)

type MockServer struct {
	srv *grpc.Server
}

// node network service mock server implementation
type serverApi struct {
}

func (s *serverApi) UpdateContainer(ctx context.Context, req *containersprotocol.UpdateContainerRequest) (*containersprotocol.UpdateContainerResponse, error) {

	bmReq := req.Container.Type.(*containersprotocol.Container_BareMetalContainer)

	if bmReq.BareMetalContainer.LogicalContainerId == nil {
		return nil, fmt.Errorf("LogicalContainerId is empty")
	}

	if bmReq.BareMetalContainer.BladeId == nil {
		return nil, fmt.Errorf("BladeId is empty")
	}

	if bmReq.BareMetalContainer.Id == nil {
		return nil, fmt.Errorf("PhysicalId is empty")
	}

	return &containersprotocol.UpdateContainerResponse{}, nil
}

func (s *serverApi) DeleteContainer(ctx context.Context, req *containersprotocol.DeleteContainerRequest) (*containersprotocol.DeleteContainerResponse, error) {

	if req.ContainerIdentifier.LogicalContainerId == nil {
		return nil, fmt.Errorf("LogicalContainerId is empty")
	}

	if req.ContainerIdentifier.ContainerId == nil {
		return nil, fmt.Errorf("PhysicalId is empty")
	}

	return &containersprotocol.DeleteContainerResponse{}, nil
}

func (s *serverApi) GetLegacyContainerConfiguration(
	context.Context,
	*containersprotocol.GetLegacyContainerConfigurationRequest) (*containersprotocol.GetLegacyContainerConfigurationResponse, error) {
	return nil, nil
}

func NewMockServer() *MockServer {
	return &MockServer{
		srv: grpc.NewServer(),
	}
}

func (s *MockServer) StartGrpcServer(port int) {

	endpoint := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Errorf("aznmmockserver: failed to listen at endpoint %s with error %v", endpoint, err)
	}

	containersprotocol.RegisterNetworkContainerServiceServer(s.srv, &serverApi{})
	if err := s.srv.Serve(lis); err != nil {
		log.Errorf("aznmmockserver: failed to serve at port %d with error %s", port, err)
	}
}

func (s *MockServer) StopGrpcServer() {
	if s.srv == nil {
		fmt.Printf("s.srv is nil \n")
	}
	s.srv.Stop()
}
