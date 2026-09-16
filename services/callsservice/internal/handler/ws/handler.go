package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"messengermax/callsservice/internal/entity"
	"messengermax/pkg/jwt"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

// Signaling payloads the frontend sends to the server.
type inbound struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type invitePayload struct {
	CalleeID uint64 `json:"callee_id"`
	ChatID   uint64 `json:"chat_id"`
	CallType string `json:"call_type"`
}

type acceptPayload struct {
	CallID uint64 `json:"call_id"`
}

type rejectPayload struct {
	CallID uint64 `json:"call_id"`
	Reason string `json:"reason"`
}

type endPayload struct {
	CallID uint64 `json:"call_id"`
	Reason string `json:"reason"`
}

type signalPayload struct {
	CallID uint64 `json:"call_id"`
	Data   string `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// CallStateManager exposes the parts of the gRPC call service that the
// signaling controller needs. It is implemented by service.Server.
type CallStateManager interface {
	StartCallForSignaling(ctx context.Context, callerID, calleeID uint, chatID uint, callType entity.CallType) (*entity.Call, error)
	AcceptCallForSignaling(ctx context.Context, callID, userID uint) (*entity.Call, error)
	EndCallForSignaling(ctx context.Context, callID, userID uint, reason string) (*entity.Call, error)
	FindActiveForSignaling(ctx context.Context, userID uint) (*entity.Call, error)
}

type callRoom struct {
	callID uint64
	caller uint64
	callee uint64
}

// Controller manages signaling connections: auth, upgrade, call-state
// transitions and peer-to-peer message relay.
type Controller struct {
	jwtManager *jwt.Manager
	hub        *Hub
	manager    CallStateManager

	roomsMu sync.RWMutex
	rooms   map[uint64]*callRoom // callID -> room
}

func NewController(jwtManager *jwt.Manager, h *Hub, manager CallStateManager) *Controller {
	return &Controller{jwtManager: jwtManager, hub: h, manager: manager, rooms: make(map[uint64]*callRoom)}
}

// Handle upgrades the request and runs the connection's read loop.
func (c *Controller) Handle(gctx *gin.Context) {
	token := gctx.Query("token")
	if token == "" {
		gctx.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}
	userID, err := c.jwtManager.ValidateToken(token)
	if err != nil {
		gctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(gctx.Writer, gctx.Request, nil)
	if err != nil {
		log.Printf("calls: upgrade failed: %v", err)
		return
	}

	c.hub.Register(uint(userID), conn)
	c.hub.SendToUser(uint(userID), WSMessage{Type: "CALL_CONNECTED", Payload: gin.H{"user_id": userID}})
	go c.pingLoop(conn)
	c.readLoop(conn, uint(userID))
}

func (c *Controller) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			_ = conn.Close()
			return
		}
	}
}

func (c *Controller) readLoop(conn *websocket.Conn, userID uint) {
	conn.SetReadLimit(16 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg inbound
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		c.dispatch(userID, conn, msg)
	}
	c.hub.Unregister(userID, conn)
	_ = conn.Close()
	// If the user was mid-call, tear the call down server-side.
	c.cleanupOnDisconnect(userID)
}

func (c *Controller) dispatch(userID uint, conn *websocket.Conn, msg inbound) {
	ctx := context.Background()
	switch msg.Type {
	case "CALL_INVITE":
		c.handleInvite(ctx, userID, msg.Payload)
	case "CALL_ACCEPT":
		c.handleAccept(ctx, userID, msg.Payload)
	case "CALL_REJECT":
		c.handleReject(ctx, userID, msg.Payload)
	case "CALL_END":
		c.handleEnd(ctx, userID, msg.Payload)
	case "CALL_SDP", "CALL_ICE":
		c.handleRelay(userID, msg.Payload)
	}
}

func (c *Controller) handleInvite(ctx context.Context, callerID uint, raw json.RawMessage) {
	var p invitePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.CalleeID == 0 {
		c.hub.SendToUser(callerID, WSMessage{Type: "CALL_ERROR", Payload: gin.H{"error": "invalid invite"}})
		return
	}
	callType := entity.CallTypeAudio
	if p.CallType == string(entity.CallTypeVideo) {
		callType = entity.CallTypeVideo
	}
	call, err := c.manager.StartCallForSignaling(ctx, callerID, uint(p.CalleeID), uint(p.ChatID), callType)
	if err != nil {
		c.hub.SendToUser(callerID, WSMessage{Type: "CALL_ERROR", Payload: gin.H{"error": err.Error()}})
		return
	}

	c.roomsMu.Lock()
	c.rooms[uint64(call.ID)] = &callRoom{callID: uint64(call.ID), caller: uint64(callerID), callee: p.CalleeID}
	c.roomsMu.Unlock()

	// Notify the caller that the call is now ringing.
	c.hub.SendToUser(callerID, WSMessage{Type: "CALL_INVITE_ACK", Payload: callPayload(call)})

	if !c.hub.IsOnline(uint(p.CalleeID)) {
		c.failCall(ctx, call, "unreachable", callerID, uint(p.CalleeID))
		return
	}
	// Ring the callee.
	c.hub.SendToUser(uint(p.CalleeID), WSMessage{
		Type: "CALL_RINGING",
		Payload: gin.H{
			"call_id":   call.ID,
			"caller_id": callerID,
			"call_type": call.CallType,
		},
	})
}

func (c *Controller) handleAccept(ctx context.Context, userID uint, raw json.RawMessage) {
	var p acceptPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.CallID == 0 {
		return
	}
	call, err := c.manager.AcceptCallForSignaling(ctx, uint(p.CallID), userID)
	if err != nil {
		return
	}
	c.broadcastRoom(uint64(call.ID), WSMessage{Type: "CALL_ACCEPTED", Payload: callPayload(call)})
}

func (c *Controller) handleReject(ctx context.Context, userID uint, raw json.RawMessage) {
	var p rejectPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.CallID == 0 {
		return
	}
	reason := p.Reason
	if reason == "" {
		reason = "rejected"
	}
	call, err := c.manager.EndCallForSignaling(ctx, uint(p.CallID), userID, reason)
	if err != nil {
		return
	}
	c.broadcastRoom(uint64(call.ID), WSMessage{Type: "CALL_REJECTED", Payload: callPayload(call)})
	c.removeRoom(uint64(call.ID))
}

func (c *Controller) handleEnd(ctx context.Context, userID uint, raw json.RawMessage) {
	var p endPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.CallID == 0 {
		return
	}
	reason := p.Reason
	if reason == "" {
		reason = "normal"
	}
	call, err := c.manager.EndCallForSignaling(ctx, uint(p.CallID), userID, reason)
	if err != nil {
		return
	}
	c.broadcastRoom(uint64(call.ID), WSMessage{Type: "CALL_ENDED", Payload: callPayload(call)})
	c.removeRoom(uint64(call.ID))
}

// handleRelay forwards a signaling message (SDP offer/answer or ICE
// candidate) from one peer to the other peer within the same call.
func (c *Controller) handleRelay(userID uint, raw json.RawMessage) {
	var p signalPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.CallID == 0 {
		return
	}
	room := c.roomOf(p.CallID)
	if room == nil {
		return
	}
	var target uint64
	switch userID {
	case uint(room.caller):
		target = room.callee
	case uint(room.callee):
		target = room.caller
	default:
		return
	}
	// Re-wrap the payload so the receiver gets the exact original message.
	var full inbound
	if err := json.Unmarshal(raw, &full); err != nil {
		return
	}
	c.hub.SendToUser(uint(target), WSMessage{Type: full.Type, Payload: full.Payload})
}

// failCall ends a ringing call that cannot be delivered (callee offline).
func (c *Controller) failCall(ctx context.Context, call *entity.Call, reason string, callerID, calleeID uint) {
	// End from the caller's side; the callee never saw the call.
	endReason := reason
	if endReason == "unreachable" {
		endReason = "no_answer"
	}
	if _, err := c.manager.EndCallForSignaling(ctx, call.ID, callerID, endReason); err != nil {
		log.Printf("calls: fail call %d: %v", call.ID, err)
	}
	c.hub.SendToUser(callerID, WSMessage{Type: "CALL_ENDED", Payload: callPayload(call)})
	c.removeRoom(uint64(call.ID))
}

// cleanupOnDisconnect ends any active call the user is part of when their
// signaling connection drops.
func (c *Controller) cleanupOnDisconnect(userID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	call, err := c.manager.FindActiveForSignaling(ctx, userID)
	if err != nil || call == nil {
		return
	}
	if _, err := c.manager.EndCallForSignaling(ctx, call.ID, userID, "disconnect"); err != nil {
		return
	}
	c.broadcastRoom(uint64(call.ID), WSMessage{Type: "CALL_ENDED", Payload: callPayload(call)})
	c.removeRoom(uint64(call.ID))
}

func (c *Controller) broadcastRoom(callID uint64, msg WSMessage) {
	room := c.roomOf(callID)
	if room == nil {
		return
	}
	c.hub.SendToUsers([]uint{uint(room.caller), uint(room.callee)}, msg)
}

// NotifyCallEnded pushes an authoritative CALL_ENDED to both participants
// (used by the ringing-timeout sweeper). The room is cleared as well.
func (c *Controller) NotifyCallEnded(call *entity.Call) {
	c.broadcastRoom(uint64(call.ID), WSMessage{Type: "CALL_ENDED", Payload: callPayload(call)})
	c.removeRoom(uint64(call.ID))
}

func (c *Controller) roomOf(callID uint64) *callRoom {
	c.roomsMu.RLock()
	defer c.roomsMu.RUnlock()
	return c.rooms[callID]
}

func (c *Controller) removeRoom(callID uint64) {
	c.roomsMu.Lock()
	delete(c.rooms, callID)
	c.roomsMu.Unlock()
}

func callPayload(call *entity.Call) gin.H {
	return gin.H{
		"call_id":        call.ID,
		"caller_id":      call.CallerID,
		"callee_id":      call.CalleeID,
		"chat_id":        call.ChatID,
		"call_type":      call.CallType,
		"status":         call.Status,
		"started_at_ms":  call.StartedAtMs,
		"accepted_at_ms": call.AcceptedAtMs,
		"ended_at_ms":    call.EndedAtMs,
		"duration_ms":    call.DurationMs,
		"end_reason":     call.EndReason,
	}
}
