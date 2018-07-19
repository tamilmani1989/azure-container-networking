package extensionhandler

import (
	"encoding/json"
)

// States of Handler
const (
	Transistioning = "transitioning"
	Warning        = "warning"
	Success        = "success"
	Error          = "error"
)

// Status codes of Handler
const (
	PassCode = iota
	WarningCode
	FailCode
)

type HandlerConfig struct {
	RuntimeSettings []RuntimeSettings `json:"runtimeSettings"`
}

type RuntimeSettings struct {
	HandlerSettings HandlerSetttings `json:"handlerSettings"`
}

type HandlerSetttings struct {
	PublicSettings PublicSettings `json:"publicSettings"`
}

type PublicSettings struct {
	Plugins []Plugins `json:"plugins"`
}

type Plugins struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

type ServiceTunnelConfig struct {
	PortsToOpen []int `json:"portsToOpen"`
}

type Location struct {
	ConfigFolder  string `json:"configFolder"`
	LogFolder     string `json:"logFolder"`
	HeartbeatFile string `json:"heartbeatFile"`
	StatusFolder  string `json:"statusFolder"`
}

type HandlerEnvironment struct {
	Version float32  `json:"version"`
	Name    string   `json:"name"`
	Env     Location `json:"handlerEnvironment"`
}

type HandlerStatus struct {
	Version      string        `json:"version"`
	TimestampUTC string        `json:"timestampUTC"`
	Status       StatusMessage `json:"status"`
}

type StatusMessage struct {
	Name                     string           `json:"name"`
	Operation                string           `json:"operation"`
	Configurationappliedtime string           `json:"configurationAppliedTime"`
	Status                   string           `json:"status"` //Transitioning, error, success and warning
	Code                     int              `json:"code"`
	Message                  Message          `json: "message"`
	FormattedMessage         FormattedMessage `json:"formattedMessage"`
	SubStatus                []SubStatus      `json:"substatus"`
}

type Message struct {
	ID     string   `json:"id"`
	Params []string `json:"params"`
}

type FormattedMessage struct {
	Lang    string `json:"lang"`
	Message string `json:"message"`
}

type SubStatus struct {
	Name             string           `json:"name"`
	Status           string           `json:"status"`
	Code             int              `json:"code"`
	Message          Message          `json: "message"`
	FormattedMessage FormattedMessage `json:"formattedMessage"`
}
