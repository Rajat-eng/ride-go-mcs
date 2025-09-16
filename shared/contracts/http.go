package contracts

// APIResponse is the response structure for the API.
// type APIResponse struct {
// 	Data  any       `json:"data,omitempty"`
// 	Error *APIError `json:"error,omitempty"`
// }

// // APIError is the error structure for the API.
// type APIError struct {
// 	Code    string `json:"code"`
// 	Message string `json:"message"`
// }

type ErrorResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
