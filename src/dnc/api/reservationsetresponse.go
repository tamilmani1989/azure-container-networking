package api

// ReservationResponse - Struct to hold reservation response.
type ReservationSetResponse struct {
	NetworkID                  string                `json:"NetworkID"`
	SubnetName                 string                `json:"SubnetName"`
	ReservationSetID           string                `json:"ReservationSetID"`
	RequestedReservationCount  int                   `json:"RequestedReservationCount"`
	SuccessfulReservationCount int                   `json:"SuccessfulReservationCount"`
	TotalReservationCount      int                   `json:"TotalReservationCount"`
	Reservations               []ReservationResponse `json:"Reservations"`
	RequestStatus              RequestStatus         `json:"RequestStatus"`
}
