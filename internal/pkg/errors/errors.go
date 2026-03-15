package errors

import "fmt"

type Code string

const (
	CodeNotFound       Code = "NOT_FOUND"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeBadRequest     Code = "BAD_REQUEST"
	CodeInternal       Code = "INTERNAL"
	CodeRateLimit      Code = "RATE_LIMIT"
	CodeUpstreamError  Code = "UPSTREAM_ERROR"
	CodeSessionExpired Code = "SESSION_EXPIRED"
)

type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
