package error_middleware

// JSONErrors
// @Description JSONErrors is a dto for cases one or more errors occurred during request handling
type JSONErrors struct {
	Errors []internalError `json:"errors"`
}

// JSONInfo
// @Description JSONInfo is a dto for cases request has no data to return.
// @Description JSONInfo is used to provide some information about what action happened
type JSONInfo struct {
	Info string `json:"info"`
}
