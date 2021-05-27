package api

// ReservationSetRequest - Struct to hold reservationset request.
type ReservationSetRequest struct {
	SubnetName       string `json:"SubnetName"`
	NumReservations  int    `json:"NumReservations"`
}

// Validate - Validates the reservationset request
func (r *ReservationSetRequest) Validate() *JsonErr {
	if r.SubnetName == "" {
		return BadRequest(InvalidIPSubnet, "subnet name cannot be empty")
	}

	if r.NumReservations <= 0 {
		return BadRequestf(InvalidNumReservations, "Invalid value for number of reservations: %d", r.NumReservations)
	}

	return nil
}
