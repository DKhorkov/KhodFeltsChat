// Package handlers Khod Felts Chat API
//
// Documentation for REST API of KFC
//
//	Schemes: http
//	BasePath: /
//	Version: 1.0.0
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
// swagger:meta
package handlers

// OK message returned as an HTTP Status Code
// swagger:response OK
type OK struct {
	// Description of the situation
	// in: body
	StatusCode int
}

// BadRequest message returned as an HTTP Status Code
// swagger:response BadRequest
type BadRequest struct {
	// Description of the situation
	// in: body
	StatusCode int
}

// NotFound message returned as an HTTP Status Code
// swagger:response NotFound
type NotFound struct {
	// Description of the situation
	// in: body
	StatusCode int
}

// InternalServerError message returned as an HTTP Status Code
// swagger:response InternalServerError
type InternalServerError struct {
	// Description of the situation
	// in: body
	StatusCode int
}

// AlreadyExistsError message returned as an HTTP Status Code
// swagger:response AlreadyExistsError
type AlreadyExistsError struct {
	// Description of the situation
	// in: body
	StatusCode int
}
