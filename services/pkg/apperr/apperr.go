package apperr

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
)

type Error struct {
	Code    codes.Code
	Message string
}

func (e *Error) Error() string { return e.Message }

func New(code codes.Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

func Errorf(code codes.Code, format string, a ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, a...)}
}

func CodeOf(err error) codes.Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return codes.Internal
}

func MsgOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}
