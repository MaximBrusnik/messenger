package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"messengermax/pkg/apperr"

	"messengermax/callsservice/internal/entity"
	"messengermax/callsservice/internal/service"
	pb "messengermax/proto/gen/calls"
)

type Server struct {
	pb.UnimplementedCallServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) StartCall(ctx context.Context, req *pb.StartCallRequest) (*pb.CallSession, error) {
	if req.CalleeId == 0 {
		return nil, status.Error(codes.InvalidArgument, "callee required")
	}
	call, err := s.svc.StartCall(ctx, uint(req.CallerId), uint(req.CalleeId), 0, fromPbCallType(req.CallType))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(call), nil
}

func (s *Server) AcceptCall(ctx context.Context, req *pb.AcceptCallRequest) (*pb.CallSession, error) {
	call, err := s.svc.AcceptCall(ctx, uint(req.CallId), uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(call), nil
}

func (s *Server) EndCall(ctx context.Context, req *pb.EndCallRequest) (*pb.CallSession, error) {
	call, err := s.svc.EndCall(ctx, uint(req.CallId), uint(req.UserId), req.Reason)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(call), nil
}

func (s *Server) GetActiveCall(ctx context.Context, req *pb.GetActiveCallRequest) (*pb.CallSession, error) {
	call, err := s.svc.GetActiveCall(ctx, uint(req.UserId))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(call), nil
}

func (s *Server) GetCallHistory(ctx context.Context, req *pb.GetCallHistoryRequest) (*pb.CallsResponse, error) {
	calls, err := s.svc.GetCallHistory(ctx, uint(req.UserId), int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, toGRPCError(err)
	}
	res := &pb.CallsResponse{Calls: make([]*pb.CallSession, 0, len(calls))}
	for i := range calls {
		res.Calls = append(res.Calls, toProto(&calls[i]))
	}
	return res, nil
}

func toProto(call *entity.Call) *pb.CallSession {
	return &pb.CallSession{
		Id:           uint64(call.ID),
		CallerId:     uint64(call.CallerID),
		CalleeId:     uint64(call.CalleeID),
		CallType:     toPbCallType(call.CallType),
		Status:       toPbStatus(call.Status),
		StartedAtMs:  call.StartedAtMs,
		AcceptedAtMs: call.AcceptedAtMs,
		EndedAtMs:    call.EndedAtMs,
		DurationMs:   call.DurationMs,
		EndReason:    call.EndReason,
	}
}

func toPbCallType(t entity.CallType) pb.CallType {
	switch t {
	case entity.CallTypeVideo:
		return pb.CallType_CALL_TYPE_VIDEO
	default:
		return pb.CallType_CALL_TYPE_AUDIO
	}
}

func fromPbCallType(t pb.CallType) entity.CallType {
	if t == pb.CallType_CALL_TYPE_VIDEO {
		return entity.CallTypeVideo
	}
	return entity.CallTypeAudio
}

func toPbStatus(s entity.CallStatus) pb.CallStatus {
	switch s {
	case entity.CallStatusRinging:
		return pb.CallStatus_CALL_STATUS_RINGING
	case entity.CallStatusActive:
		return pb.CallStatus_CALL_STATUS_ACTIVE
	case entity.CallStatusEnded:
		return pb.CallStatus_CALL_STATUS_ENDED
	case entity.CallStatusMissed:
		return pb.CallStatus_CALL_STATUS_MISSED
	case entity.CallStatusRejected:
		return pb.CallStatus_CALL_STATUS_REJECTED
	case entity.CallStatusCancelled:
		return pb.CallStatus_CALL_STATUS_CANCELLED
	default:
		return pb.CallStatus_CALL_STATUS_UNSPECIFIED
	}
}

func toGRPCError(err error) error {
	return status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
}
