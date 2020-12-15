package networkcontainer

import (
	"dnc/datamodel"
	"errors"
	"fmt"
)

type requestSet map[string]reservationRequest

func (r requestSet) put(req reservationRequest) {
	r[req.reservation.ReservationID] = req
}

func (r requestSet) len() int {
	return len(r)
}

func (r requestSet) complete(completedReservations []datamodel.ReservationConfiguration) {
	for _, reservation := range completedReservations {
		resReq, ok := r[reservation.ReservationID]
		if ok {
			resReq.responseCh <- response{reservation: reservation}
			delete(r, reservation.ReservationID)
		}
	}

	// flush if any left. signals a programming error in batch processing
	if len(r) > 0 {
		r.flushWithError(errors.New("batch completed without handling this request"))
	}
}

func (r requestSet) flushWithError(err error) {
	fmt.Printf("flushing all requests with error: %v\n", err)
	for _, req := range r {
		req.responseCh <- response{err: err}
		delete(r, req.reservation.ReservationID)
	}
}

func (r requestSet) removeWithError(resID string, err error) {
	resReq, ok := r[resID]
	if ok {
		resReq.responseCh <- response{err: err}
		delete(r, resID)
	}
}

func (r requestSet) getAllReservations() []datamodel.ReservationConfiguration {
	var out []datamodel.ReservationConfiguration
	for _, req := range r {
		out = append(out, req.reservation)
	}
	return out
}

func (r requestSet) getAllValidTBDAllocIDs() map[string][]string {
	reservationIDToTBDs := make(map[string][]string)
	for _, req := range r {
		resID := req.reservation.ReservationID
		// verify its still in reservation, no-op when len is 0
		for _, allocID := range req.toBeDeletedIPIDs {
			if _, ok := req.reservation.SecondaryAllocations[allocID]; ok {
				reservationIDToTBDs[resID] = append(reservationIDToTBDs[resID], allocID)
			}
		}
	}
	return reservationIDToTBDs
}
