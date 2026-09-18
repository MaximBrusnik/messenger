package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pb "messengermax/proto/gen/user"
)

func (g *Gateway) GetSettings(c *gin.Context) {
	resp, err := g.usr.GetSettings(ctx(), &pb.GetSettingsRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"show_online_status": resp.ShowOnlineStatus,
		"last_seen_privacy":  resp.LastSeenPrivacy,
		"avatar_privacy":     resp.AvatarPrivacy,
		"sound_enabled":      resp.SoundEnabled,
	}})
}

func (g *Gateway) UpdateSettings(c *gin.Context) {
	var req struct {
		ShowOnlineStatus *bool   `json:"show_online_status"`
		LastSeenPrivacy  *string `json:"last_seen_privacy"`
		AvatarPrivacy    *string `json:"avatar_privacy"`
		SoundEnabled     *bool   `json:"sound_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	r := &pb.UpdateSettingsRequest{UserId: userID(c)}
	if req.ShowOnlineStatus != nil {
		r.ShowOnlineStatus = *req.ShowOnlineStatus
	}
	if req.LastSeenPrivacy != nil {
		r.LastSeenPrivacy = *req.LastSeenPrivacy
	}
	if req.AvatarPrivacy != nil {
		r.AvatarPrivacy = *req.AvatarPrivacy
	}
	if req.SoundEnabled != nil {
		r.SoundEnabled = req.SoundEnabled
	}
	if _, err := g.usr.UpdateSettings(ctx(), r); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РќР°СЃС‚СЂРѕР№РєРё РѕР±РЅРѕРІР»РµРЅС‹"})
}

func (g *Gateway) GetAllUsers(c *gin.Context) {
	resp, err := g.usr.GetAllUsers(ctx(), &pb.GetAllUsersRequest{ExcludeId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Users))
	for _, u := range resp.Users {
		out = append(out, userJSON(u, u.Id == userID(c)))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) SearchUsers(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РїР°СЂР°РјРµС‚СЂ q РѕР±СЏР·Р°С‚РµР»РµРЅ"})
		return
	}
	resp, err := g.usr.SearchUsers(ctx(), &pb.SearchUsersRequest{Query: q, ExcludeId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Users))
	for _, u := range resp.Users {
		out = append(out, userJSON(u, u.Id == userID(c)))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) GetUser(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID"})
		return
	}
	me := userID(c)
	resp, err := g.usr.GetProfile(ctx(), &pb.GetProfileRequest{UserId: id, ViewerId: me})
	if err != nil {
		HTTPError(c, err)
		return
	}
	// legacy: third-party viewers never see the email
	self := id == me
	if !self {
		resp.Email = ""
		resp.EmailVerified = false
	}
	c.JSON(http.StatusOK, gin.H{"data": userJSON(resp, self)})
}

func (g *Gateway) GetContacts(c *gin.Context) {
	resp, err := g.usr.GetContacts(ctx(), &pb.GetContactsRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Users))
	for _, u := range resp.Users {
		out = append(out, userJSON(u, u.Id == userID(c)))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) AddContact(c *gin.Context) {
	var req struct {
		UserID uint64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	_, err := g.usr.AddContact(ctx(), &pb.AddContactRequest{UserId: userID(c), ContactId: req.UserID})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РљРѕРЅС‚Р°РєС‚ РґРѕР±Р°РІР»РµРЅ"})
}
