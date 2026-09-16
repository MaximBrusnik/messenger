package api

import (
	"context"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	pbchat "messengermax/proto/gen/chat"
	pbuser "messengermax/proto/gen/user"
)

// userJSON renders a UserProfile as the legacy REST UserResponse.
// self indicates the viewer is the owner (full fields are exposed).
func userJSON(p *pbuser.UserProfile, self bool) gin.H {
	h := gin.H{
		"id":       p.Id,
		"username": p.Username,
		"avatar":   p.Avatar,
		"status":   p.Status,
	}
	if self {
		h["email"] = p.Email
		h["email_verified"] = p.EmailVerified
		h["is_bot"] = p.IsBot
		h["is_admin"] = p.IsAdmin
	} else {
		if p.IsBot || p.IsAdmin {
			h["is_bot"] = p.IsBot
			h["is_admin"] = p.IsAdmin
		}
	}
	if s := ts(p.CreatedAt); s != "" {
		h["created_at"] = s
	}
	if s := ts(p.LastLogin); s != "" {
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

// profilesByID fetches many profiles and indexes them by user id.
func (g *Gateway) profilesByID(ctx context.Context, ids []uint64) map[uint64]*pbuser.UserProfile {
	seen := make(map[uint64]bool)
	var uniq []uint64
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		uniq = append(uniq, id)
	}
	out := make(map[uint64]*pbuser.UserProfile, len(uniq))
	if len(uniq) == 0 {
		return out
	}
	resp, err := g.usr.GetProfilesBulk(ctx, &pbuser.GetProfilesBulkRequest{UserIds: uniq})
	if err != nil {
		return out
	}
	for _, p := range resp.Users {
		out[p.Id] = p
	}
	return out
}

// reactionJSON renders a chat reaction like the legacy API.
func reactionJSON(r *pbchat.Reaction, profiles map[uint64]*pbuser.UserProfile) gin.H {
	username := r.Username
	if p, ok := profiles[r.UserId]; ok && username == "" {
		username = p.Username
	}
	h := gin.H{
		"message_id": r.MessageId,
		"user_id":    r.UserId,
		"reaction":   r.Reaction,
	}
	if username != "" {
		h["username"] = username
	}
	if s := ts(r.CreatedAt); s != "" {
		h["created_at"] = s
	}
	return h
}

// messageJSON builds the full legacy MessageResponse, enriching the sender
// and reaction usernames with profile data.
func (g *Gateway) messageJSON(ctx context.Context, m *pbchat.Message) gin.H {
	profileIDs := []uint64{m.SenderId}
	if m.IsForwarded && m.ForwardedFromSenderId > 0 {
		profileIDs = append(profileIDs, m.ForwardedFromSenderId)
	}
	profiles := g.profilesByID(ctx, profileIDs)
	sender := gin.H{}
	if p, ok := profiles[m.SenderId]; ok {
		sender = userJSON(p, p.Id == m.SenderId)
	}

	reactions := make([]gin.H, 0, len(m.Reactions))
	for _, r := range m.Reactions {
		reactions = append(reactions, reactionJSON(r, profiles))
	}

	h := gin.H{
		"id":         m.Id,
		"chat_id":    m.ChatId,
		"sender_id":  m.SenderId,
		"text":       m.Text,
		"is_read":    m.IsRead,
		"created_at": ts(m.CreatedAt),
		"edited":     m.Edited,
		"sender":     sender,
	}
	if s := ts(m.ReadAt); s != "" {
		h["read_at"] = s
	}
	if s := ts(m.EditedAt); s != "" {
		h["edited_at"] = s
	}
	if len(reactions) > 0 {
		h["reactions"] = reactions
	}
	if m.IsForwarded {
		h["is_forwarded"] = true
		ff := gin.H{
			"id":      m.ForwardedFromSenderId,
			"chat_id": m.ForwardedFromChatId,
		}
		if p, ok := profiles[m.ForwardedFromSenderId]; ok {
			ff["username"] = p.Username
			ff["avatar"] = p.Avatar
		}
		h["forwarded_from"] = ff
	}
	if m.SystemType != "" {
		h["system_type"] = m.SystemType
	}
	if m.AttachmentType != "" {
		h["attachment_type"] = m.AttachmentType
		h["attachment_url"] = m.AttachmentUrl
		if m.AttachmentName != "" {
			h["attachment_name"] = m.AttachmentName
		}
		if m.AttachmentSize > 0 {
			h["attachment_size"] = m.AttachmentSize
		}
	}
	return h
}

// chatJSON builds the legacy ChatResponse. For private chats the display
// name/avatar are taken from the other participant.
func (g *Gateway) chatJSON(ctx context.Context, viewer uint64, c *pbchat.Chat) gin.H {
	pids := make([]uint64, 0, len(c.Participants))
	for _, p := range c.Participants {
		pids = append(pids, p.UserId)
	}
	profiles := g.profilesByID(ctx, pids)

	participants := make([]gin.H, 0, len(c.Participants))
	for _, p := range c.Participants {
		if prof, ok := profiles[p.UserId]; ok {
			participants = append(participants, userJSON(prof, prof.Id == viewer))
		}
	}

	name := c.Name
	avatar := c.Avatar
	if c.IsFavorites {
		name = "Избранное"
		avatar = ""
	} else if c.Type == "private" && len(profiles) > 0 {
		var other *pbuser.UserProfile
		for _, p := range profiles {
			if p.Id != viewer {
				other = p
				break
			}
		}
		if other != nil {
			name = other.Username
			avatar = other.Avatar
		}
	}

	h := gin.H{
		"id":           c.Id,
		"name":         name,
		"type":         c.Type,
		"created_at":   ts(c.CreatedAt),
		"updated_at":   ts(c.UpdatedAt),
		"is_favorites": c.IsFavorites,
	}
	if avatar != "" {
		h["avatar"] = avatar
	}
	if len(participants) > 0 {
		h["participants"] = participants
	}
	if c.Unread > 0 {
		h["unread"] = c.Unread
	}
	if c.LastMessage != nil {
		h["last_message"] = lastMessageJSON(c.LastMessage)
	}
	if c.PinnedMessage != nil {
		h["pinned_message"] = g.messageJSON(ctx, c.PinnedMessage)
	}
	return h
}

// lastMessageJSON is the reduced message shape used by chat lists.
func lastMessageJSON(m *pbchat.Message) gin.H {
	h := gin.H{
		"id":         m.Id,
		"chat_id":    m.ChatId,
		"sender_id":  m.SenderId,
		"text":       m.Text,
		"is_read":    m.IsRead,
		"created_at": ts(m.CreatedAt),
	}
	if m.SystemType != "" {
		h["system_type"] = m.SystemType
	}
	if m.AttachmentType != "" {
		h["attachment_type"] = m.AttachmentType
		h["attachment_url"] = m.AttachmentUrl
	}
	return h
}

// sortReactions keeps output deterministic.
func sortReactions(list []*pbchat.Reaction) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].UserId != list[j].UserId {
			return list[i].UserId < list[j].UserId
		}
		return strings.Compare(list[i].Reaction, list[j].Reaction) < 0
	})
}
