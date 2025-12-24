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

// Conflict message returned when entity already exists or operation already done
// swagger:response Conflict
type Conflict struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// Unauthorized message returned when no information about current User
// swagger:response Unauthorized
type Unauthorized struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// Forbidden message returned when user has no rights
// swagger:response Forbidden
type Forbidden struct {
	// HTTP Status Code
	// in: header
	StatusCode int

	// Description of the situation
	// in: body
	Error string
}

// SeeOther message returned when redirect to another URL was made
// swagger:response SeeOther
type SeeOther struct {
	// HTTP Status Code
	// in: header
	StatusCode int
}

// NoContent message returned when there is no payload in Response Body
// swagger:response NoContent
type NoContent struct {
	// HTTP Status Code
	// in: header
	StatusCode int
}
