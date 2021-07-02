// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/MadariClient"
	sdn "dnc/microsoft/azure/sdncontracts"
	"fmt"
	"google.golang.org/protobuf/proto"
)

func GetNICProgrammingStatus(reservationID string) (string, error) {
	partitionKey := generateAznmStatusPartitionKey(reservationID)
	relativePath := networkInterfaceStatusPath

	for pubsubAzId, publisher := range publisherContext.publishers {
		networkInterfaceStatus, err := getNetworkInterfaceStatusWithPublisher(publisher, partitionKey, relativePath)
		if err != nil {
			return "", err
		}

		logDebugf("OnResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Data: %v",
			pubsubAzId, partitionKey, relativePath, networkInterfaceStatus)
		return networkInterfaceStatus.ProgrammingStatus.String(), nil
	}

	return "", fmt.Errorf("PubSubAzID not found from publisher")
}

func getNetworkInterfaceStatus(pubsubAzId, partitionKey, relativePath string) (sdn.NetworkInterfaceStatus, error) {
	result, err := getData(pubsubAzId, partitionKey, relativePath)
	if err != nil {
		return sdn.NetworkInterfaceStatus{}, fmt.Errorf("getData(%s, %s) failed with error: %+v", partitionKey, relativePath, err)
	}

	return deserializeNetworkInterfaceStatus(pubsubAzId, partitionKey, relativePath, result)
}

func getNetworkInterfaceStatusWithPublisher(publisher *publisher, partitionKey, relativePath string) (sdn.NetworkInterfaceStatus, error) {
	result, err := getDataWithPublisher(publisher, partitionKey, relativePath)
	if err != nil {
		return sdn.NetworkInterfaceStatus{}, fmt.Errorf("getData(%s, %s) failed with error: %+v", partitionKey, relativePath, err)
	}

	return deserializeNetworkInterfaceStatus(publisher.pubsubAzId, partitionKey, relativePath, result)
}

func deserializeNetworkInterfaceStatus(
	pubsubAzId,
	partitionKey,
	relativePath string,
	result MadariClient.MadariResult) (sdn.NetworkInterfaceStatus, error) {

	networkInterfaceStatus := sdn.NetworkInterfaceStatus{}
	err := proto.Unmarshal(result.Data, &networkInterfaceStatus)
	if err != nil {
		msg := fmt.Sprintf("onResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Error: %v",
			pubsubAzId, partitionKey, relativePath, err)
		logErrorf(msg)
		return sdn.NetworkInterfaceStatus{}, fmt.Errorf(msg)
	}

	logDebugf("onResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Data: %v",
		pubsubAzId, partitionKey, relativePath, networkInterfaceStatus)

	return networkInterfaceStatus, nil
}
