package response

var (
	ErrInvalidRequestFormat = ErrorResponse{
		Status:  "error",
		Error:   "invalid_request",
		Details: "Invalid request format",
	}

	ErrAuthenticationFailed = ErrorResponse{
		Status: "error",
		Error:  "authentication_failed",
	}

	ErrInvalidRegisterRequest = ErrorResponse{
		Status:  "error",
		Error:   "invalid_register_request",
		Details: "Invalid registration data",
	}

	ErrUserAlreadyExists = ErrorResponse{
		Status:  "error",
		Error:   "user_already_exists",
		Details: "User with this email already exists",
	}
)
