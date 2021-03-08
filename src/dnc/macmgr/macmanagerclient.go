package macmgr

import (
	"bytes"
	"dnc/api"
	"dnc/configuration"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
)

type MacManagerClient interface {
	AllocateMac(allocationId string) (*MacAllocatorResponse, *api.JsonErr)
	DeallocateMac(allocationId string) (*MacAllocatorResponse, *api.JsonErr)
}

type MacManagerConnection struct {
	HttpClient        *http.Client
	MacManagerSetting *configuration.MacManagerSettings
}

type MacAllocatorRequest struct {
	AllocationId string
	Type         string
	Tag          string
}

type MacAllocatorResponse struct {
	AllocationId string `json:"AllocationId"`
	MacAddress   string `json:"MacAddress"`
	IsSuccess    bool   `json:"IsSuccess"`
	ErrorMessage string `json:"ErrorMessage"`
	ShouldRetry  bool   `json:"ShouldRetry"`
}

const (
	defaultContentType = "application/json; charset=UTF-8"
)

func (m *MacManagerConnection) AllocateMac(allocationId string) (*MacAllocatorResponse, *api.JsonErr) {
	allocateMacURL := m.MacManagerSetting.AllocateMacURL

	allocateMacRequest := &MacAllocatorRequest{
		AllocationId: allocationId,
		// Below two values are empty for now. We have them for extensibility and future use.
		Tag:  "",
		Type: "",
	}

	var allocateMacRequestBody bytes.Buffer
	if encodeErr := json.NewEncoder(&allocateMacRequestBody).Encode(allocateMacRequest); encodeErr != nil {
		msg := fmt.Sprintf("Allocate MAC failed. AllocationId = %s, Reason = encoding error: %s",
			allocationId,
			encodeErr.Error())
		return nil, api.BadRequest(api.MacAllocationFailed, msg)
	}

	w, respErr := m.HttpClient.Post(allocateMacURL, defaultContentType, &allocateMacRequestBody)

	if respErr != nil {
		msg := fmt.Sprintf("Allocate MAC failed. AllocationId = %s, Reason = Post to URL %s failed with error %s",
			allocationId,
			allocateMacURL,
			respErr.Error())
		return nil, api.InternalServerError(api.MacAllocationFailed, msg)
	}

	defer w.Body.Close()

	var resp MacAllocatorResponse
	respErr = utils.DecodeResponse(w, &resp)
	if respErr != nil {
		msg := fmt.Sprintf("Allocate MAC failed. AllocationId = %s, Reason = Failed to decode response with error %s",
			allocationId,
			respErr.Error())
		return nil, api.InternalServerError(api.MacAllocationFailed, msg)
	}

	if w.StatusCode == http.StatusBadRequest {
		return nil, api.BadRequest(api.MacAllocationFailed, resp.ErrorMessage)
	} else if w.StatusCode == http.StatusInternalServerError {
		return nil, api.InternalServerError(api.MacAllocationFailed, resp.ErrorMessage)
	} else {
		return &resp, nil
	}
}

func (m *MacManagerConnection) DeallocateMac(allocationId string) (*MacAllocatorResponse, *api.JsonErr) {
	deallocateMacURL := m.MacManagerSetting.DeallocateMacURL

	deallocateMacRequest := &MacAllocatorRequest{
		AllocationId: allocationId,
		// Below two values are empty for now. We have them for extensibility and future use.
		Tag:  "",
		Type: "",
	}

	var deallocateMacRequestBody bytes.Buffer
	if encodeErr := json.NewEncoder(&deallocateMacRequestBody).Encode(deallocateMacRequest); encodeErr != nil {
		msg := fmt.Sprintf("Deallocate MAC failed. AllocationId = %s, Reason = encoding error: %s",
			allocationId,
			encodeErr.Error())

		return nil, api.BadRequest(api.MacDeallocationFailed, msg)
	}

	w, respErr := m.HttpClient.Post(deallocateMacURL, defaultContentType, &deallocateMacRequestBody)

	if respErr != nil {
		msg := fmt.Sprintf("Deallocate MAC failed. AllocationId = %s, Reason = Post to URL %s failed with error %s",
			allocationId,
			deallocateMacURL,
			respErr.Error())
		return nil, api.InternalServerError(api.MacDeallocationFailed, msg)
	}

	defer w.Body.Close()

	var resp MacAllocatorResponse
	respErr = utils.DecodeResponse(w, &resp)
	if respErr != nil {
		msg := fmt.Sprintf("Deallocate MAC failed. AllocationId = %s, Reason =  Failed to decode response with error %s",
			allocationId,
			respErr.Error())
		return nil, api.InternalServerError(api.MacDeallocationFailed, msg)
	}

	if w.StatusCode == http.StatusBadRequest {
		return nil, api.BadRequest(api.MacAllocationFailed, resp.ErrorMessage)
	} else if w.StatusCode == http.StatusInternalServerError {
		return nil, api.InternalServerError(api.MacAllocationFailed, resp.ErrorMessage)
	} else {
		return &resp, nil
	}
}
