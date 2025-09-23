package errors

import "fmt"

type ErrorCode int

const (
	ErrorCodeNotFound ErrorCode = iota + 1
	ErrorCodeDuplicate
	ErrorCodeValidation
	ErrorCodeInternal
	ErrorCodeBadRequest
)

type ServiceError struct {
	Op      string
	Code    ErrorCode
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

func NewNotFoundError(op, message string) *ServiceError {
	return &ServiceError{
		Op:      op,
		Code:    ErrorCodeNotFound,
		Message: message,
	}
}

func NewValidationError(op, message string, err error) *ServiceError {
	return &ServiceError{
		Op:      op,
		Code:    ErrorCodeValidation,
		Message: message,
		Err:     err,
	}
}

func NewDuplicateError(op, message string) *ServiceError {
	return &ServiceError{
		Op:      op,
		Code:    ErrorCodeDuplicate,
		Message: message,
	}
}

func NewInternalError(op, message string, err error) *ServiceError {
	return &ServiceError{
		Op:      op,
		Code:    ErrorCodeInternal,
		Message: message,
		Err:     err,
	}
}
