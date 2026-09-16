package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	pb "messengermax/proto/gen/chat"
)

func (g *Gateway) GetChats(c *gin.Context) {
	resp, err := g.chat.GetUserChats(ctx(), &pb.GetUserChatsRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Chats))
	for _, ch := range resp.Chats {
		out = append(out, g.chatJSON(ctx(), userID(c), ch))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) CreateChat(c *gin.Context) {
	var req struct {
		UserID uint64 `json:"user_id" binding:"required"`
		Name   string `json:"name"`
		Type   string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РќРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	chatType := req.Type
	if chatType == "" {
		chatType = "private"
	}
	resp, err := g.chat.CreateChat(ctx(), &pb.CreateChatRequest{
		UserId:         userID(c),
		ParticipantIds: []uint64{req.UserID},
		Name:           req.Name,
		Type:           chatType,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Р§Р°С‚ СЃРѕР·РґР°РЅ", "data": g.chatJSON(ctx(), userID(c), resp)})
}

func (g *Gateway) GetChat(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	resp, err := g.chat.GetChatByID(ctx(), &pb.GetChatRequest{UserId: userID(c), ChatId: id})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": g.chatJSON(ctx(), userID(c), resp)})
}

func (g *Gateway) GetMessages(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := g.chat.GetMessages(ctx(), &pb.GetMessagesRequest{
		UserId: userID(c), ChatId: id, Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		out = append(out, g.messageJSON(ctx(), m))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) SendMessage(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	var req struct {
		Content        string `json:"content"`
		AttachmentType string `json:"attachment_type"`
		AttachmentURL  string `json:"attachment_url"`
		AttachmentName string `json:"attachment_name"`
		AttachmentSize *int64 `json:"attachment_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	if req.Content == "" && req.AttachmentURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РўРµРєСЃС‚ СЃРѕРѕР±С‰РµРЅРёСЏ РёР»Рё РІР»РѕР¶РµРЅРёРµ РѕР±СЏР·Р°С‚РµР»СЊРЅРѕ"})
		return
	}
	size := int64(0)
	if req.AttachmentSize != nil {
		size = *req.AttachmentSize
	}
	resp, err := g.chat.SendMessage(ctx(), &pb.SendMessageRequest{
		UserId: userID(c), ChatId: id, Content: req.Content,
		AttachmentType: req.AttachmentType, AttachmentUrl: req.AttachmentURL,
		AttachmentName: req.AttachmentName, AttachmentSize: size,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ РѕС‚РїСЂР°РІР»РµРЅРѕ", "data": g.messageJSON(ctx(), resp)})
}

func (g *Gateway) EditMessage(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РєРѕРЅС‚РµРЅС‚ РѕР±СЏР·Р°С‚РµР»РµРЅ"})
		return
	}
	resp, err := g.chat.EditMessage(ctx(), &pb.EditMessageRequest{
		UserId: userID(c), ChatId: chatID, MessageId: msgID, Content: req.Content,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ РѕС‚СЂРµРґР°РєС‚РёСЂРѕРІР°РЅРѕ", "data": g.messageJSON(ctx(), resp)})
}

func (g *Gateway) DeleteMessage(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	_, err = g.chat.DeleteMessage(ctx(), &pb.DeleteMessageRequest{
		UserId: userID(c), ChatId: chatID, MessageId: msgID,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ СѓРґР°Р»РµРЅРѕ"})
}

func (g *Gateway) MarkAsRead(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	_, err = g.chat.MarkAsRead(ctx(), &pb.MarkAsReadRequest{UserId: userID(c), ChatId: chatID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёСЏ РѕС‚РјРµС‡РµРЅС‹ РєР°Рє РїСЂРѕС‡РёС‚Р°РЅРЅС‹Рµ"})
}

func (g *Gateway) PinMessage(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	_, err = g.chat.PinMessage(ctx(), &pb.PinMessageRequest{UserId: userID(c), ChatId: chatID, MessageId: msgID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ Р·Р°РєСЂРµРїР»РµРЅРѕ"})
}

func (g *Gateway) UnpinMessage(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	_, err = g.chat.UnpinMessage(ctx(), &pb.UnpinMessageRequest{UserId: userID(c), ChatId: chatID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ РѕС‚РєСЂРµРїР»РµРЅРѕ"})
}

func (g *Gateway) DeleteChat(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	_, err = g.chat.DeleteChat(ctx(), &pb.DeleteChatRequest{UserId: userID(c), ChatId: chatID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Р§Р°С‚ СѓРґР°Р»С‘РЅ"})
}

func (g *Gateway) ForwardMessage(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	resp, err := g.chat.ForwardMessage(ctx(), &pb.ForwardMessageRequest{
		UserId: userID(c), ChatId: chatID, MessageId: msgID,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЎРѕРѕР±С‰РµРЅРёРµ РїРµСЂРµСЃР»Р°РЅРѕ", "data": g.messageJSON(ctx(), resp)})
}

func (g *Gateway) GetOrCreateAIChat(c *gin.Context) {
	resp, err := g.chat.GetOrCreateAIChat(ctx(), &pb.GetOrCreateAIChatRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": g.chatJSON(ctx(), userID(c), resp)})
}

func (g *Gateway) GetOrCreateFavoritesChat(c *gin.Context) {
	resp, err := g.chat.GetOrCreateFavoritesChat(ctx(), &pb.GetOrCreateFavoritesChatRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": g.chatJSON(ctx(), userID(c), resp)})
}

func (g *Gateway) ArchiveChat(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	_, err = g.chat.ArchiveChat(ctx(), &pb.ArchiveChatRequest{UserId: userID(c), ChatId: chatID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Р§Р°С‚ Р°СЂС…РёРІРёСЂРѕРІР°РЅ"})
}

func (g *Gateway) UnarchiveChat(c *gin.Context) {
	chatID, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID С‡Р°С‚Р°"})
		return
	}
	_, err = g.chat.UnarchiveChat(ctx(), &pb.UnarchiveChatRequest{UserId: userID(c), ChatId: chatID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Р§Р°С‚ СЂР°Р·Р°СЂС…РёРІРёСЂРѕРІР°РЅ"})
}

func (g *Gateway) GetArchivedChats(c *gin.Context) {
	resp, err := g.chat.GetUserArchivedChats(ctx(), &pb.GetUserChatsRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Chats))
	for _, ch := range resp.Chats {
		out = append(out, g.chatJSON(ctx(), userID(c), ch))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) GetReactions(c *gin.Context) {
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	resp, err := g.chat.GetReactions(ctx(), &pb.GetReactionsRequest{MessageId: msgID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	sortReactions(resp.Reactions)
	var ids []uint64
	for _, r := range resp.Reactions {
		ids = append(ids, r.UserId)
	}
	profiles := g.profilesByID(ctx(), ids)
	out := make([]gin.H, 0, len(resp.Reactions))
	for _, r := range resp.Reactions {
		out = append(out, reactionJSON(r, profiles))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) AddReaction(c *gin.Context) {
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	var req struct {
		Reaction string `json:"reaction" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "СЂРµР°РєС†РёСЏ РѕР±СЏР·Р°С‚РµР»СЊРЅР°"})
		return
	}
	resp, err := g.chat.AddReaction(ctx(), &pb.AddReactionRequest{
		UserId: userID(c), MessageId: msgID, Reaction: req.Reaction,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Р РµР°РєС†РёСЏ РґРѕР±Р°РІР»РµРЅР°", "data": reactionJSON(resp, nil)})
}

func (g *Gateway) RemoveReaction(c *gin.Context) {
	msgID, err := parseID(c, "msgId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID СЃРѕРѕР±С‰РµРЅРёСЏ"})
		return
	}
	reaction := c.Query("reaction")
	if reaction == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "СЂРµР°РєС†РёСЏ РѕР±СЏР·Р°С‚РµР»СЊРЅР°"})
		return
	}
	_, err = g.chat.RemoveReaction(ctx(), &pb.RemoveReactionRequest{
		UserId: userID(c), MessageId: msgID, Reaction: reaction,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Р РµР°РєС†РёСЏ СѓРґР°Р»РµРЅР°"})
}
