package server

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type OffsetOutOfRangeError struct {
	Offset uint64
}

func (e OffsetOutOfRangeError) GRPCStatus() *status.Status {
	st := status.New(404, fmt.Sprintf("offset out of range: %d", e.Offset))
	msg := fmt.Sprintf(
		"The requested offset is outside of the log: %d",
		e.Offset,
	)
	d := &errdetails.LocalizedMessage{
		Locale:  "en-US",
		Message: msg,
	}
	std, err := st.WithDetails(d)
	if err != nil {
		return st
	}
	return std
}

func (e OffsetOutOfRangeError) Error() string {
	return e.GRPCStatus().Err().Error()
}
