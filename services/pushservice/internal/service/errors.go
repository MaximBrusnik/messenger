package service

//TODO вынести отсюда

import (
	"google.golang.org/grpc/codes"

	"messengermax/pkg/apperr"
)

func errInvalid(msg string) error          { return apperr.New(codes.InvalidArgument, msg) }
func errInternal(msg string) error         { return apperr.New(codes.Internal, msg) }
func errNotFound(msg string) error         { return apperr.New(codes.NotFound, msg) }
func errPermissionDenied(msg string) error { return apperr.New(codes.PermissionDenied, msg) }
func errAlreadyExists(msg string) error    { return apperr.New(codes.AlreadyExists, msg) }
func errFailedPrecondition(msg string) error {
	return apperr.New(codes.FailedPrecondition, msg)
}
func errUnavailable(msg string) error {
	return apperr.New(codes.Unavailable, msg)
}
func errResourceExhausted(msg string) error {
	return apperr.New(codes.ResourceExhausted, msg)
}
