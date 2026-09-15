package service

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"messengermax/calls/internal/entity"
	"messengermax/calls/internal/repo"
	"messengermax/pkg/nats"
	pb "messengermax/proto/gen/calls"
)

// RingTimeout is how long a call may stay in "ringing" before the callee
// is considered to have missed it.
const RingTimeout = 30 * time.Second

type Server struct {
	pb.UnimplementedCallServiceServer
	callRepo repo.CallRepository
	producer *nats.Producer
}

func NewServer(callRepo repo.CallRepository, producer *nats.Producer) *Server {
	return &Server{callRepo: callRepo, producer: producer}
}

func (s *Server) StartCall(ctx context.Context, req *pb.StartCallRequest) (*pb.CallSession, error) {
	if req.CalleeId == 0 {
		return nil, status.Error(codes.InvalidArgument, "callee required")
	}
	if req.CallerId == req.CalleeId {
		return nil, status.Error(codes.InvalidArgument, "нельзя позвонить самому себе")
	}
	if _, err := s.callRepo.FindActiveForUser(uint(req.CallerId)); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "у вас уже есть активный звонок")
	}
	if _, err := s.callRepo.FindActiveForUser(uint(req.CalleeId)); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "собеседник уже в звонке")
	}
	now := time.Now().UnixMilli()
	call := &entity.Call{
		CallerID:    uint(req.CallerId),
		CalleeID:    uint(req.CalleeId),
		CallType:    fromPbCallType(req.CallType),
		Status:      entity.CallStatusRinging,
		StartedAtMs: now,
		EndReason:   "timeout", // default for a ringing call that is never accepted
		TimeToDieAt: now + RingTimeout.Milliseconds(),
	}
	if err := s.callRepo.Create(call); err != nil {
		return nil, status.Error(codes.Internal, "не удалось создать звонок")
	}
	return toProto(call), nil
}

// StartCallForSignaling is the entry point used by the WebSocket controller
// when a caller sends CALL_INVITE. Returns the created call or a stderr-like
// error message the controller relays to the caller.
func (s *Server) StartCallForSignaling(ctx context.Context, callerID, calleeID uint, chatID uint, callType entity.CallType) (*entity.Call, error) {
	call, err := s.StartCall(ctx, &pb.StartCallRequest{
		CallerId: uint64(callerID),
		CalleeId: uint64(calleeID),
		CallType: toPbCallType(callType),
	})
	if err != nil {
		return nil, err
	}
	e := fromProto(call)
	if chatID != 0 {
		e.ChatID = chatID
		_ = s.callRepo.Update(e)
	}
	return e, nil
}

func (s *Server) AcceptCallForSignaling(ctx context.Context, callID, userID uint) (*entity.Call, error) {
	call, err := s.AcceptCall(ctx, &pb.AcceptCallRequest{CallId: uint64(callID), UserId: uint64(userID)})
	if err != nil {
		return nil, err
	}
	return fromProto(call), nil
}

func (s *Server) AcceptCall(ctx context.Context, req *pb.AcceptCallRequest) (*pb.CallSession, error) {
	if req.CallId == 0 {
		return nil, status.Error(codes.InvalidArgument, "call_id required")
	}
	call, err := s.callRepo.FindByID(uint(req.CallId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "звонок не найден")
	}
	if call.Status != entity.CallStatusRinging {
		return nil, status.Error(codes.FailedPrecondition, "звонок уже завершён")
	}
	// Only the callee may accept, but allow either participant for robustness.
	call.Status = entity.CallStatusActive
	if req.UserId != 0 {
		call.AcceptedAtMs = time.Now().UnixMilli()
		call.TimeToDieAt = 0
	}
	if err := s.callRepo.Update(call); err != nil {
		return nil, status.Error(codes.Internal, "не удалось обновить звонок")
	}
	return toProto(call), nil
}

func (s *Server) EndCall(ctx context.Context, req *pb.EndCallRequest) (*pb.CallSession, error) {
	if req.CallId == 0 {
		return nil, status.Error(codes.InvalidArgument, "call_id required")
	}
	call, err := s.callRepo.FindByID(uint(req.CallId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "звонок не найден")
	}
	if call.Status == entity.CallStatusEnded ||
		call.Status == entity.CallStatusCancelled ||
		call.Status == entity.CallStatusRejected ||
		call.Status == entity.CallStatusMissed {
		return toProto(call), nil
	}
	now := time.Now().UnixMilli()
	endedAt := now
	if call.AcceptedAtMs > 0 {
		call.DurationMs = now - call.AcceptedAtMs
	} else {
		// Ringing call that was never accepted: no duration.
		call.DurationMs = 0
	}

	reason := req.Reason
	switch {
	case reason == "rejected":
		call.Status = entity.CallStatusRejected
	case reason == "no_answer" || reason == "unreachable" || reason == "timeout":
		call.Status = entity.CallStatusMissed
	case call.AcceptedAtMs == 0:
		call.Status = entity.CallStatusCancelled
	default:
		call.Status = entity.CallStatusEnded
	}
	call.EndedAtMs = endedAt
	call.EndReason = reason
	if err := s.callRepo.Update(call); err != nil {
		return nil, status.Error(codes.Internal, "не удалось завершить звонок")
	}
	s.publishCallEnded(call)
	return toProto(call), nil
}

// publishCallEnded emits the call-ended event so the chat service can write
// a system message into the originating chat.
func (s *Server) publishCallEnded(call *entity.Call) {
	if s.producer == nil || call.ChatID == 0 {
		return
	}
	s.producer.Publish(nats.TopicCallEnded, itoa(uint64(call.ID)), nats.EventCallEnded{
		CallID:      int64(call.ID),
		ChatID:      int64(call.ChatID),
		CallerID:    int64(call.CallerID),
		CalleeID:    int64(call.CalleeID),
		CallType:    string(call.CallType),
		Status:      string(call.Status),
		EndReason:   call.EndReason,
		StartedAtMs: call.StartedAtMs,
		EndedAtMs:   call.EndedAtMs,
		DurationMs:  call.DurationMs,
	})
}

func (s *Server) EndCallForSignaling(ctx context.Context, callID, userID uint, reason string) (*entity.Call, error) {
	call, err := s.EndCall(ctx, &pb.EndCallRequest{
		CallId: uint64(callID), UserId: uint64(userID), Reason: reason,
	})
	if err != nil {
		return nil, err
	}
	return fromProto(call), nil
}

func (s *Server) GetActiveCall(ctx context.Context, req *pb.GetActiveCallRequest) (*pb.CallSession, error) {
	call, err := s.callRepo.FindActiveForUser(uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "нет активных звонков")
	}
	return toProto(call), nil
}

// FindActiveForSignaling is used by the WebSocket controller to tear down a
// call when a participant's signaling connection drops.
func (s *Server) FindActiveForSignaling(ctx context.Context, userID uint) (*entity.Call, error) {
	return s.callRepo.FindActiveForUser(userID)
}

// FindExpiredRinging returns ringing calls that exceeded their ring timeout.
func (s *Server) FindExpiredRinging(nowMs int64) ([]entity.Call, error) {
	return s.callRepo.FindExpiredRinging(nowMs)
}

// ExpireRingingCall ends a ringing call that nobody accepted (ring timeout).
// It routes through EndCall so the call-ended event is published.
func (s *Server) ExpireRingingCall(ctx context.Context, call *entity.Call) (*entity.Call, error) {
	ended, err := s.EndCall(ctx, &pb.EndCallRequest{
		CallId: uint64(call.ID),
		UserId: uint64(call.CallerID),
		Reason: "timeout",
	})
	if err != nil {
		return nil, err
	}
	return fromProto(ended), nil
}

func (s *Server) GetCallHistory(ctx context.Context, req *pb.GetCallHistoryRequest) (*pb.CallsResponse, error) {
	calls, err := s.callRepo.History(uint(req.UserId), int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить историю звонков")
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

func itoa(n uint64) string {
	return strconv.FormatUint(n, 10)
}

func fromProto(call *pb.CallSession) *entity.Call {
	return &entity.Call{
		ID:           uint(call.Id),
		CallerID:     uint(call.CallerId),
		CalleeID:     uint(call.CalleeId),
		CallType:     fromPbCallType(call.CallType),
		Status:       fromPbStatus(call.Status),
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

func fromPbStatus(s pb.CallStatus) entity.CallStatus {
	switch s {
	case pb.CallStatus_CALL_STATUS_RINGING:
		return entity.CallStatusRinging
	case pb.CallStatus_CALL_STATUS_ACTIVE:
		return entity.CallStatusActive
	case pb.CallStatus_CALL_STATUS_ENDED:
		return entity.CallStatusEnded
	case pb.CallStatus_CALL_STATUS_MISSED:
		return entity.CallStatusMissed
	case pb.CallStatus_CALL_STATUS_REJECTED:
		return entity.CallStatusRejected
	case pb.CallStatus_CALL_STATUS_CANCELLED:
		return entity.CallStatusCancelled
	default:
		return entity.CallStatusRinging
	}
}
