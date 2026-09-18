package rest

import (
	"context"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	pbchat "messengermax/proto/gen/chat"
	pbuser "messengermax/proto/gen/user"
)

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

func (g *Gateway) profilesByID(ctx context.Context, viewer uint64, ids []uint64) map[uint64]*pbuser.UserProfile {
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
	resp, err := g.usr.GetProfilesBulk(ctx, &pbuser.GetProfilesBulkRequest{UserIds: uniq, ViewerId: viewer})
	if err != nil {
		return out
	}
	for _, p := range resp.Users {
		out[p.Id] = p
	}
	return out
}

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

func (g *Gateway) messageJSON(ctx context.Context, viewer uint64, m *pbchat.Message) gin.H {
	return g.messageJSONWithProfiles(ctx, viewer, m, g.profilesByID(ctx, viewer, messageProfileIDs([]*pbchat.Message{m})))
}

func (g *Gateway) messageJSONWithProfiles(ctx context.Context, viewer uint64, m *pbchat.Message, profiles map[uint64]*pbuser.UserProfile) gin.H {
	sender := gin.H{}
	if p, ok := profiles[m.SenderId]; ok {
		sender = userJSON(p, p.Id == viewer)
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
	if m.ReplyToMessageId > 0 {
		rt := gin.H{"id": m.ReplyToMessageId}
		if m.ReplyTo != nil {
			rt["sender_id"] = m.ReplyTo.SenderId
			rt["text"] = m.ReplyTo.Text
			if p, ok := profiles[m.ReplyTo.SenderId]; ok {
				rt["username"] = p.Username
				rt["avatar"] = p.Avatar
			}
			if m.ReplyTo.SystemType != "" {
				rt["system_type"] = m.ReplyTo.SystemType
			}
			if m.ReplyTo.AttachmentType != "" {
				rt["attachment_type"] = m.ReplyTo.AttachmentType
				rt["attachment_name"] = m.ReplyTo.AttachmentName
			}
		} else {
			rt["deleted"] = true
		}
		h["reply_to"] = rt
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

func (g *Gateway) chatJSON(ctx context.Context, viewer uint64, c *pbchat.Chat) gin.H {
	return g.chatJSONWithProfiles(ctx, viewer, c, g.profilesByID(ctx, viewer, chatProfileIDs([]*pbchat.Chat{c})))
}

func (g *Gateway) chatJSONWithProfiles(ctx context.Context, viewer uint64, c *pbchat.Chat, profiles map[uint64]*pbuser.UserProfile) gin.H {
	participants := make([]gin.H, 0, len(c.Participants))
	for _, p := range c.Participants {
		if prof, ok := profiles[p.UserId]; ok {
			participants = append(participants, userJSON(prof, prof.Id == viewer))
		}
	}

	name := c.Name
	avatar := c.Avatar
	if c.IsFavorites {
		name = "РР·Р±СЂР°РЅРЅРѕРµ"
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
		h["pinned_message"] = g.messageJSONWithProfiles(ctx, viewer, c.PinnedMessage, profiles)
	}
	return h
}

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

func sortReactions(list []*pbchat.Reaction) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].UserId != list[j].UserId {
			return list[i].UserId < list[j].UserId
		}
		return strings.Compare(list[i].Reaction, list[j].Reaction) < 0
	})
}

func messageProfileIDs(msgs []*pbchat.Message) []uint64 {
	seen := make(map[uint64]bool)
	var ids []uint64
	add := func(id uint64) {
		if id == 0 || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, m := range msgs {
		if m == nil {
			continue
		}
		add(m.SenderId)
		if m.IsForwarded && m.ForwardedFromSenderId > 0 {
			add(m.ForwardedFromSenderId)
		}
		if m.ReplyTo != nil && m.ReplyTo.SenderId > 0 {
			add(m.ReplyTo.SenderId)
		}
	}
	return ids
}

func chatProfileIDs(chats []*pbchat.Chat) []uint64 {
	seen := make(map[uint64]bool)
	var ids []uint64
	for _, c := range chats {
		if c == nil {
			continue
		}
		for _, p := range c.Participants {
			if p == nil || p.UserId == 0 || seen[p.UserId] {
				continue
			}
			seen[p.UserId] = true
			ids = append(ids, p.UserId)
		}
	}
	return ids
}

func reactionProfileIDs(rs []*pbchat.Reaction) []uint64 {
	seen := make(map[uint64]bool)
	var ids []uint64
	for _, r := range rs {
		if r == nil || r.UserId == 0 || seen[r.UserId] {
			continue
		}
		seen[r.UserId] = true
		ids = append(ids, r.UserId)
	}
	return ids
}
