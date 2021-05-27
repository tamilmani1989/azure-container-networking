package networkcontainer

import (
	"dnc/datamodel"
	"errors"
	"fmt"
)

type requestSet map[string]reservationRequest

func (r requestSet) put(req reservationRequest) {
	r[req.reservation.Configuration.ReservationID] = req
}

func (r requestSet) len() int {
	return len(r)
}

func (r requestSet) complete(completedReservations []datamodel.ReservationRow) {
	for _, reservation := range completedReservations {
		resReq, ok := r[reservation.Configuration.ReservationID]
		if ok {
			resReq.responseCh <- response{reservation: reservation}
			delete(r, reservation.Configuration.ReservationID)
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
		delete(r, req.reservation.Configuration.ReservationID)
	}
}

func (r requestSet) removeWithError(resID string, err error) {
	resReq, ok := r[resID]
	if ok {
		resReq.responseCh <- response{err: err}
		delete(r, resID)
	}
}

func (r requestSet) getAllReservations() []datamodel.ReservationRow {
	var out []datamodel.ReservationRow
	for _, req := range r {
		out = append(out, req.reservation)
	}
	return out
}

func (r requestSet) getAllValidTBDAllocIDs() map[string][]string {
	reservationIDToTBDs := make(map[string][]string)
	for _, req := range r {
		resID := req.reservation.Configuration.ReservationID
		// verify its still in reservation, no-op when len is 0
		for _, allocID := range req.toBeDeletedIPIDs {
			if _, ok := req.reservation.Configuration.SecondaryAllocations[allocID]; ok {
				reservationIDToTBDs[resID] = append(reservationIDToTBDs[resID], allocID)
			}
		}
	}
	return reservationIDToTBDs
}
