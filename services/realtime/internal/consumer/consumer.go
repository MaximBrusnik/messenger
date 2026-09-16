package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/pkg/nats"
	"messengermax/pkg/redis"
	pbuser "messengermax/proto/gen/user"
	"messengermax/realtime/internal/hub"
)

// Consumer translates Kafka chat events into WebSocket pushes.
type Consumer struct {
	hub     *hub.Hub
	rc      *redis.Client
	users   pbuser.UserServiceClient
	timeout time.Duration
}

func New(h *hub.Hub, rc *redis.Client, users pbuser.UserServiceClient) *Consumer {
	return &Consumer{hub: h, rc: rc, users: users, timeout: 2 * time.Second}
}

// HandleUserEvent applies online/offline transitions published by any
// realtime instance to the shared Redis state.
func (c *Consumer) HandleUserEvent(ctx context.Context, value []byte) {
	var e nats.EventUserStatus
	if err := json.Unmarshal(value, &e); err != nil {
		return
	}
	if e.Online {
		_ = c.rc.SetOnline(ctx, uint(e.UserID))
	} else {
		_ = c.rc.SetOffline(ctx, uint(e.UserID))
	}
}

// Handle is registered as the kafka consumer callback.
func (c *Consumer) Handle(topic string, key string, value []byte) {
	switch topic {
	case nats.TopicMessageCreated:
		var e nats.EventMessageCreated
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		ids := uintIDs(e.Participants)
		c.hub.SendToUsers(ids, hub.WSMessage{Type: "NEW_MESSAGE", Payload: gin.H{"message": c.messageJSON(e.Message)}})

	case nats.TopicMessageEdited:
		var e nats.EventMessageEdited
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "MESSAGE_EDITED", Payload: gin.H{"message": c.messageJSON(e.Message)}})

	case nats.TopicMessageDeleted:
		var e nats.EventMessageDeleted
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "MESSAGE_DELETED", Payload: e})

	case nats.TopicMessagePinned:
		var e nats.EventMessagePinned
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "MESSAGE_PINNED", Payload: gin.H{"chat_id": e.ChatID, "message": c.messageJSON(e.Message)}})

	case nats.TopicMessageUnpinned:
		var e nats.EventMessagePinned
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "MESSAGE_UNPINNED", Payload: e})

	case nats.TopicReactionAdded:
		var e nats.EventReactionAdded
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "REACTION_ADDED", Payload: gin.H{"chat_id": e.ChatID, "message_id": e.MessageID, "reaction": e.Reaction}})

	case nats.TopicReactionRemoved:
		var e nats.EventReactionRemoved
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "REACTION_REMOVED", Payload: e})

	case nats.TopicReadReceived:
		var e nats.EventMessagesRead
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "MESSAGES_READ", Payload: e})

	case nats.TopicChatDeleted:
		var e nats.EventChatDeleted
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		c.hub.Broadcast(hub.WSMessage{Type: "CHAT_DELETED", Payload: e})
	}
}

// messageJSON builds the full legacy-style message payload (same shape as
// the REST gateway) so the frontend can render it without refetching.
func (c *Consumer) messageJSON(dto nats.MessageDTO) gin.H {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	profiles := c.profilesByID(ctx, c.messageUserIDs(dto))

	h := gin.H{
		"id":         dto.ID,
		"chat_id":    dto.ChatID,
		"sender_id":  dto.SenderID,
		"text":       dto.Text,
		"is_read":    dto.IsRead,
		"created_at": msToTime(dto.CreatedAtMs),
		"edited":     dto.Edited,
		"sender":     c.senderJSON(profiles[uint64(dto.SenderID)], dto.SenderID),
	}
	if dto.ReadAtMs > 0 {
		h["read_at"] = msToTime(dto.ReadAtMs)
	}
	if dto.EditedAtMs > 0 {
		h["edited_at"] = msToTime(dto.EditedAtMs)
	}
	if dto.SystemType != "" {
		h["system_type"] = dto.SystemType
	}
	if dto.IsForwarded {
		h["is_forwarded"] = true
		ff := gin.H{
			"id":      dto.ForwardedFromSenderID,
			"chat_id": dto.ForwardedFromChatID,
		}
		if p, ok := profiles[uint64(dto.ForwardedFromSenderID)]; ok {
			ff["username"] = p.Username
			ff["avatar"] = p.Avatar
		}
		h["forwarded_from"] = ff
	}
	if dto.AttachmentType != "" {
		h["attachment_type"] = dto.AttachmentType
		h["attachment_url"] = dto.AttachmentURL
		if dto.AttachmentName != "" {
			h["attachment_name"] = dto.AttachmentName
		}
		if dto.AttachmentSize > 0 {
			h["attachment_size"] = dto.AttachmentSize
		}
	}
	if len(dto.Reactions) > 0 {
		reactions := make([]gin.H, 0, len(dto.Reactions))
		for _, r := range dto.Reactions {
			rh := gin.H{
				"message_id": r.MessageID,
				"user_id":    r.UserID,
				"reaction":   r.Reaction,
			}
			username := r.Username
			if p, ok := profiles[uint64(r.UserID)]; ok && username == "" {
				username = p.Username
			}
			if username != "" {
				rh["username"] = username
			}
			if r.CreatedAtMs > 0 {
				rh["created_at"] = msToTime(r.CreatedAtMs)
			}
			reactions = append(reactions, rh)
		}
		h["reactions"] = reactions
	}
	return h
}

func (c *Consumer) messageUserIDs(dto nats.MessageDTO) []uint64 {
	seen := make(map[uint64]bool, len(dto.Reactions)+1)
	ids := make([]uint64, 0, len(dto.Reactions)+1)
	add := func(id int64) {
		if id <= 0 {
			return
		}
		u := uint64(id)
		if seen[u] {
			return
		}
		seen[u] = true
		ids = append(ids, u)
	}
	add(dto.SenderID)
	if dto.IsForwarded {
		add(dto.ForwardedFromSenderID)
	}
	for _, r := range dto.Reactions {
		add(r.UserID)
	}
	return ids
}

func (c *Consumer) profilesByID(ctx context.Context, ids []uint64) map[uint64]*pbuser.UserProfile {
	out := make(map[uint64]*pbuser.UserProfile)
	if c.users == nil || len(ids) == 0 {
		return out
	}
	resp, err := c.users.GetProfilesBulk(ctx, &pbuser.GetProfilesBulkRequest{UserIds: ids})
	if err != nil {
		return out
	}
	for _, p := range resp.Users {
		out[p.Id] = p
	}
	return out
}

func (c *Consumer) senderJSON(p *pbuser.UserProfile, fallbackID int64) gin.H {
	h := gin.H{"id": fallbackID}
	if p == nil {
		return h
	}
	h = gin.H{
		"id":       p.Id,
		"username": p.Username,
		"avatar":   p.Avatar,
		"status":   p.Status,
	}
	if p.IsBot || p.IsAdmin {
		h["is_bot"] = p.IsBot
		h["is_admin"] = p.IsAdmin
	}
	if s := tsString(p.CreatedAt); s != "" {
		h["created_at"] = s
	}
	if s := tsString(p.LastLogin); s != "" {
		h["last_login"] = s
	}
	if p.Bio != "" {
		h["bio"] = p.Bio
	}
	if p.DateOfBirth != "" {
		h["date_of_birth"] = p.DateOfBirth
	}
	return h
}

func uintIDs(ids []int64) []uint {
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			out = append(out, uint(id))
		}
	}
	return out
}

func msToTime(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05Z07:00")
}

func tsString(t *timestamppb.Timestamp) string {
	if t == nil || t.AsTime().IsZero() {
		return ""
	}
	return t.AsTime().UTC().Format("2006-01-02T15:04:05Z07:00")
}
