package schemas

// OK message returned when request was successfully processed
// swagger:response OK
type OK struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Requested data
	// in: body
	Payload string
}

// BadRequest message returned when request was incorrect
// swagger:response BadRequest
type BadRequest struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// NotFound message returned as when requested entity not found
// swagger:response NotFound
type NotFound struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// InternalServerError message returned when something not-normal happened during processing request
// swagger:response InternalServerError
type InternalServerError struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// AlreadyExistsError message returned when entity already exists
// swagger:response AlreadyExistsError
type AlreadyExistsError struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}
