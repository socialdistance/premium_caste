package response

type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func SuccessResponse(data interface{}) Response {
	return Response{
		Status: "success",
		Data:   data,
	}
}

func ErrorResponseWithDetails(err, details string) ErrorResponse {
	return ErrorResponse{
		Status:  "error",
		Error:   err,
		Details: details,
	}
}
