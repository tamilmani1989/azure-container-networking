package server

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/networkcontainer"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

// DebugRoutes provides http handler funcs for debug and devops operations on DNC
type DebugRoutes struct {
	IPAllocator *networkcontainer.IPAllocator
	Logger      loggershim.LoggerInterface
}

type rehydrateAllocationsRequest struct {
	VnetID      string
	SubnetName  string
	WorkerCount int
}

type rehydrateAllocationsResponse struct {
	Message string
}

// note: this can be an expensive call
func (d *DebugRoutes) RehydrateSubnetAllocations(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req rehydrateAllocationsRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		err := api.BadRequestf(api.JsonMarshallingFailed, "could not decode body: %v", decodeErr)
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	d.Logger.Debugf("[debug] rehydrate subnet allocations request: %+v", req)

	if !d.IPAllocator.UsePrepopulatedAllocations {
		err := api.BadRequest(api.StatusBadRequest, "DNC instance not configured for prepopulated allocations")
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	// todo: extract repo
	subnetConfig, err := datamodel.QuerySubnetConfiguration(req.VnetID, req.SubnetName)
	if err != nil {
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	_, subnet, parseErr := net.ParseCIDR(subnetConfig.IPSubnet)
	if parseErr != nil {
		err := api.InternalServerErrorf(api.StatusInternalServerError, "could not parse subnet CIDR '%s'", subnetConfig.IPSubnet)
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	allAllocs := networkcontainer.GenerateIPAllocationEntitiesForSubnet(subnet)

	currentAllocs, err := d.IPAllocator.Allocations.TryQueryAllIPAllocations(req.VnetID, req.SubnetName)
	if err != nil {
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	currAllocSet := make(map[string]struct{})
	for _, alloc := range currentAllocs {
		currAllocSet[alloc.RowKey] = struct{}{}
	}

	// determine which rows are missing
	var missing []datamodel.IPAllocationEntity
	for _, alloc := range allAllocs {
		if _, found := currAllocSet[alloc.RowKey]; !found {
			missing = append(missing, alloc)
		}
	}

	if len(missing) == 0 {
		msg := fmt.Sprintf("all allocations found for vnet %s subnet %s %s, none added", req.VnetID, req.SubnetName, subnetConfig.IPSubnet)
		d.writeResponse(w, http.StatusOK, rehydrateAllocationsResponse{Message: msg})
		return
	}

	// todo: arbitrary default and max for now
	if req.WorkerCount < 1 || req.WorkerCount > 10 {
		req.WorkerCount = 5
	}

	if err := d.IPAllocator.InsertAllocations(req.VnetID, req.SubnetName, missing, req.WorkerCount); err != nil {
		d.writeResponse(w, err.HTTPStatus, err)
		return
	}

	msg := fmt.Sprintf("added %d allocations for vnet %s subnet %s %s", len(missing), req.VnetID, req.SubnetName, subnetConfig.IPSubnet)
	d.writeResponse(w, http.StatusOK, rehydrateAllocationsResponse{Message: msg})
}

func (d *DebugRoutes) writeResponse(w http.ResponseWriter, status int, resp interface{}) {
	w.Header().Add(contentTypeStr, defaultContentType)
	if status < 100 || status > 599 {
		d.Logger.Errorf("[debug] unexpected status code %d, defaulting to 500", status)
		status = http.StatusInternalServerError
	}

	d.Logger.Debugf("[debug] response: %+v", resp)

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		d.Logger.Errorf("[debug] could not json encode response %v: %v", resp, err)
	}
}
