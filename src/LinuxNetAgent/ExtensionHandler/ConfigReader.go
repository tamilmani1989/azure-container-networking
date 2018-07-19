package extensionhandler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/Azure/azure-container-networking/log"
)

const (
	PluginName = "servicetunnel"
)

func ReadHandlerEnvironment(fileLocation string) (*HandlerEnvironment, error) {
	handlerEnv := []HandlerEnvironment{}

	readBytes, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		log.Printf("Reading HandlerEnv failed with error %v", err)
		return nil, err
	}

	if err := json.Unmarshal(readBytes, &handlerEnv); err != nil {
		log.Printf("Unmarshalling failed with err %v", err)
		return nil, err
	}

	return &handlerEnv[0], nil
}

func (handlerEnv *HandlerEnvironment) ReadHandlerConfig(settingNum int) (*HandlerConfig, error) {
	handlerConfig := &HandlerConfig{}

	handlerConfigFile := handlerEnv.Env.ConfigFolder + "/" + strconv.Itoa(settingNum) + ".settings"
	log.Printf("Read from handler config file %v", handlerConfigFile)

	readBytes, err := ioutil.ReadFile(handlerConfigFile)
	if err != nil {
		log.Printf("Reading HandlerEnv failed with error %v", err)
		return handlerConfig, err
	}

	if err := json.Unmarshal(readBytes, &handlerConfig); err != nil {
		log.Printf("Unmarshalling failed with err %v", err)
		return handlerConfig, err
	}

	return handlerConfig, nil
}

func (handlerEnv *HandlerEnvironment) ListFilesUnderConfigFolder() ([]string, error) {
	fileInfo, err := ioutil.ReadDir(handlerEnv.Env.ConfigFolder)
	if err != nil {
		log.Printf("Reading directory failed with error %v", err)
		return []string{}, err
	}

	var fileNames []string

	for _, file := range fileInfo {
		fileNames = append(fileNames, file.Name())
	}

	return fileNames, nil
}

func (handlerEnv *HandlerEnvironment) WriteStatusLog(settingsNum int, status, operation string, code int, msg string) error {
	handlerStatusArray := []HandlerStatus{}
	handlerStatus := HandlerStatus{}
	name := PluginName
	handlerStatus.Version = "1.0"
	handlerStatus.TimestampUTC = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	handlerStatus.Status.Name = name
	handlerStatus.Status.Operation = operation
	handlerStatus.Status.Status = status
	handlerStatus.Status.Code = code
	handlerStatus.Status.FormattedMessage.Lang = "en-US"
	handlerStatus.Status.FormattedMessage.Message = msg

	subStatus := SubStatus{}
	subStatus.Name = name
	subStatus.Status = status
	subStatus.Code = code
	subStatus.FormattedMessage = handlerStatus.Status.FormattedMessage

	handlerStatus.Status.SubStatus = append(handlerStatus.Status.SubStatus, subStatus)

	fileName := handlerEnv.Env.StatusFolder + "/" + strconv.Itoa(settingsNum) + ".status"

	handlerStatusArray = append(handlerStatusArray, handlerStatus)
	handlerStatusJson, err := json.Marshal(handlerStatusArray)
	if err != nil {
		log.Printf("Marshall status obj failed with error %v", err)
		return err
	}

	log.Printf("Writing status to json file %v", fileName)

	return ioutil.WriteFile(fileName, handlerStatusJson, 0644)
}

func (handlerConfig *HandlerConfig) ReadServiceTunnelConfig() (*ServiceTunnelConfig, error) {
	if len(handlerConfig.RuntimeSettings) == 0 {
		return nil, fmt.Errorf("Config Not Found")
	}

	plugins := handlerConfig.RuntimeSettings[0].HandlerSettings.PublicSettings.Plugins
	for _, plugin := range plugins {
		if plugin.Name == PluginName {
			serviceTunnelConfig := &ServiceTunnelConfig{}
			if err := json.Unmarshal(plugin.Data, &serviceTunnelConfig); err != nil {
				log.Printf("Unmarshalling failed with err %v", err)
				return nil, err
			}

			return serviceTunnelConfig, nil
		}
	}

	return nil, fmt.Errorf("ServiceTunnel plugin not found")
}
