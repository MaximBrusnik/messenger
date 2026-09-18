package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pbauth "messengermax/proto/gen/auth"
	pb "messengermax/proto/gen/user"
)

func authUserJSON(u *pbauth.User) gin.H {
	h := gin.H{
		"id":             u.Id,
		"username":       u.Username,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"is_bot":         u.IsBot,
		"is_admin":       u.IsAdmin,
		"status":         u.Status,
		"avatar":         u.Avatar,
		"created_at":     ts(u.CreatedAt),
		"last_login":     ts(u.LastLogin),
	}
	return h
}

func (g *Gateway) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "details": err.Error()})
		return
	}
	dn, dp, dfp, ip := metaFromRequest(c)
	resp, err := g.auth.Register(ctx(), &pbauth.RegisterRequest{
		Username: req.Username, Email: req.Email, Password: req.Password,
		DeviceName: dn, Platform: dp, DeviceFingerprint: dfp, Ip: ip,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	if resp.EmailVerificationRequired {
		c.JSON(http.StatusCreated, gin.H{
			"message":                     "",
			"requires_email_verification": true,
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "",
		"token":   resp.Token,
		"data":    authUserJSON(resp.User),
	})
}

func (g *Gateway) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "details": err.Error()})
		return
	}
	dn, dp, dfp, ip := metaFromRequest(c)
	resp, err := g.auth.Login(ctx(), &pbauth.LoginRequest{
		Email: req.Email, Password: req.Password,
		DeviceName: dn, Platform: dp, DeviceFingerprint: dfp, Ip: ip,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.SetCookie("access_token", resp.Token, 86400, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "РЈСЃРїРµС€РЅС‹Р№ РІС…РѕРґ",
		"token":   resp.Token,
		"data":    authUserJSON(resp.User),
	})
}

func (g *Gateway) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ""})
		return
	}
	dn, dp, dfp, ip := metaFromRequest(c)
	resp, err := g.auth.VerifyEmail(ctx(), &pbauth.VerifyEmailRequest{
		Token:      token,
		DeviceName: dn, Platform: dp, DeviceFingerprint: dfp, Ip: ip,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	if resp.Token != "" {
		c.SetCookie("access_token", resp.Token, 86400, "/", "", false, true)
	}
	c.JSON(http.StatusOK, gin.H{"message": "", "token": resp.Token})
}

func (g *Gateway) GetProfile(c *gin.Context) {
	resp, err := g.usr.GetProfile(ctx(), &pb.GetProfileRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": userJSON(resp, true)})
}

func (g *Gateway) Logout(c *gin.Context) {
	_, _ = g.auth.Logout(ctx(), &pbauth.LogoutRequest{UserId: userID(c)})
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": ""})
}

func (g *Gateway) GetDevices(c *gin.Context) {
	currentJTI, _ := c.Get("token_id")
	jti, _ := currentJTI.(string)
	resp, err := g.auth.ListSessions(ctx(), &pbauth.ListSessionsRequest{
		UserId: userID(c), CurrentJti: jti,
	})
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
			"is_current":  d.IsCurrent,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (g *Gateway) ResendVerification(c *gin.Context) {
	_, err := g.auth.ResendVerification(ctx(), &pbauth.ResendVerificationRequest{UserId: userID(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": ""})
}

func (g *Gateway) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "details": err.Error()})
		return
	}
	_, err := g.auth.ChangePassword(ctx(), &pbauth.ChangePasswordRequest{
		UserId: userID(c), OldPassword: req.OldPassword, NewPassword: req.NewPassword,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": ""})
}

func (g *Gateway) UpdateProfile(c *gin.Context) {
	var req struct {
		Username    *string `json:"username"`
		Email       *string `json:"email"`
		Avatar      *string `json:"avatar"`
		Bio         *string `json:"bio"`
		DateOfBirth *string `json:"date_of_birth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "details": err.Error()})
		return
	}
	r := &pb.UpdateProfileRequest{UserId: userID(c)}
	if req.Username != nil {
		r.Username = *req.Username
	}
	if req.Email != nil {
		r.Email = *req.Email
	}
	if req.Avatar != nil {
		if *req.Avatar == "" {
			r.ClearAvatar = true
		} else {
			r.Avatar = *req.Avatar
		}
	}
	if req.Bio != nil {
		r.Bio = *req.Bio
	}
	if req.DateOfBirth != nil {
		r.DateOfBirth = *req.DateOfBirth
	}
	resp, err := g.usr.UpdateProfile(ctx(), r)
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "", "data": userJSON(resp, true)})
}
