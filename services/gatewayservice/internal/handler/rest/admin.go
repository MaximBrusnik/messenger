package rest

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	pbauth "messengermax/proto/gen/auth"
	pbuser "messengermax/proto/gen/user"
)

// IsUserActive реализует middleware.ActiveChecker: проверяет, что аккаунт
// существует и не заблокирован (бан). Источник истины — authservice.
func (g *Gateway) IsUserActive(ctx context.Context, userID uint) (bool, bool) {
	resp, err := g.auth.CheckUserActive(ctx, &pbauth.CheckUserActiveRequest{UserId: uint64(userID)})
	if err != nil {
		return false, false
	}
	return resp.Active, resp.Found
}

func adminUserJSON(u *pbauth.AdminUser) gin.H {
	h := gin.H{
		"id":             u.Id,
		"username":       u.Username,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"is_bot":         u.IsBot,
		"is_admin":       u.IsAdmin,
		"is_active":      u.IsActive,
		"created_at":     ts(u.CreatedAt),
	}
	if s := ts(u.LastLogin); s != "" {
		h["last_login"] = s
	}
	if u.Avatar != "" {
		h["avatar"] = u.Avatar
	}
	if u.Status != "" {
		h["status"] = u.Status
	}
	return h
}

func (g *Gateway) AdminListUsers(c *gin.Context) {
	var req struct {
		Query    string `form:"q"`
		Page     uint32 `form:"page"`
		PageSize uint32 `form:"page_size"`
		IsBot    *bool  `form:"is_bot"`
		IsAdmin  *bool  `form:"is_admin"`
		Active   *bool  `form:"active"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверные параметры", "details": err.Error()})
		return
	}
	resp, err := g.auth.AdminListUsers(ctx(), &pbauth.AdminListUsersRequest{
		Query:    req.Query,
		Page:     req.Page,
		PageSize: req.PageSize,
		IsBot:    req.IsBot,
		IsAdmin:  req.IsAdmin,
		Active:   req.Active,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}

	ids := make([]uint64, 0, len(resp.Users))
	for _, u := range resp.Users {
		ids = append(ids, u.Id)
	}
	// viewer=0 — обход privacy-правил: аватар не скрывается.
	profiles := g.profilesByID(ctx(), 0, ids)

	out := make([]gin.H, 0, len(resp.Users))
	for _, u := range resp.Users {
		h := adminUserJSON(u)
		if p, ok := profiles[u.Id]; ok {
			h["avatar"] = p.Avatar
			h["online"] = p.Status == "online"
			if p.Bio != "" {
				h["bio"] = p.Bio
			}
		}
		out = append(out, h)
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": resp.Total, "page": req.Page, "page_size": req.PageSize})
}

func (g *Gateway) AdminUserStats(c *gin.Context) {
	authStats, err := g.auth.AdminUserStats(ctx(), &pbauth.Empty{})
	if err != nil {
		HTTPError(c, err)
		return
	}
	online := uint64(0)
	if uresp, err := g.usr.AdminOnlineCount(ctx(), &pbuser.Empty{}); err == nil {
		online = uresp.OnlineUsers
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"total_users":      authStats.TotalUsers,
		"active_users":     authStats.ActiveUsers,
		"banned_users":     authStats.BannedUsers,
		"unverified_users": authStats.UnverifiedUsers,
		"bots":             authStats.Bots,
		"admins":           authStats.Admins,
		"online_users":     online,
		"new_last_7days":   authStats.NewLast_7Days,
	}})
}

func (g *Gateway) AdminSetUserActive(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if id == userID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нельзя изменить статус самому себе"})
		return
	}
	var req struct {
		Active *bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Active == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "поле active обязательно"})
		return
	}
	if _, err := g.auth.AdminSetUserActive(ctx(), &pbauth.AdminSetUserActiveRequest{UserId: id, Active: *req.Active}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "статус пользователя обновлён"})
}

func (g *Gateway) AdminSetRole(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if id == userID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нельзя изменить роль самому себе"})
		return
	}
	var req struct {
		IsAdmin *bool `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.IsAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "поле is_admin обязательно"})
		return
	}
	if _, err := g.auth.AdminSetRole(ctx(), &pbauth.AdminSetRoleRequest{UserId: id, IsAdmin: *req.IsAdmin}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "роль пользователя обновлена"})
}

func (g *Gateway) AdminDeleteUser(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if id == userID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нельзя удалить самому себе"})
		return
	}
	if _, err := g.auth.AdminDeleteUser(ctx(), &pbauth.AdminDeleteUserRequest{UserId: id}); err != nil {
		HTTPError(c, err)
		return
	}
	_, _ = g.usr.AdminDeleteProfile(ctx(), &pbuser.AdminDeleteProfileRequest{UserId: id})
	c.JSON(http.StatusOK, gin.H{"message": "пользователь удалён"})
}

func (g *Gateway) AdminRevokeSessions(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if _, err := g.auth.AdminRevokeSessions(ctx(), &pbauth.AdminRevokeSessionsRequest{UserId: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "сессии завершены"})
}

func (g *Gateway) AdminUserSessions(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	resp, err := g.auth.ListSessions(ctx(), &pbauth.ListSessionsRequest{UserId: id})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		out = append(out, gin.H{
			"name":        d.Name,
			"platform":    d.Platform,
			"ip":          d.Ip,
			"first_login": ts(d.FirstLogin),
			"last_login":  ts(d.LastLogin),
			"login_count": d.LoginCount,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}
