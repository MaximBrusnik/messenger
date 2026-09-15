package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"

	pbmusic "messengermax/proto/gen/music"
	pbuser "messengermax/proto/gen/user"
)

// MusicFileProxy forwards upload/stream/download to the music-service,
// injecting the gateway's trusted identity headers.
func (g *Gateway) MusicFileProxy(c *gin.Context) {
	target, err := url.Parse("http://" + g.cfg.MusicHTTPAddr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "music-service недоступен"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	req := c.Request
	req.Header.Set("X-User-Id", itoa(userID(c)))
	req.Header.Set("X-Is-Admin", boolToStr(g.IsAdmin(c)))
	proxy.ServeHTTP(c.Writer, req)
}

func (g *Gateway) IsAdmin(c *gin.Context) bool {
	uid := userID(c)
	if uid == 0 {
		return false
	}
	if v, ok := c.Get("_is_admin"); ok {
		return v.(bool)
	}
	resp, err := g.usr.GetProfile(ctx(), &pbuser.GetProfileRequest{UserId: uid})
	admin := err == nil && resp.IsAdmin
	c.Set("_is_admin", admin)
	return admin
}

func (g *Gateway) MusicList(c *gin.Context) {
	resp, err := g.mus.List(ctx(), &pbmusic.ListRequest{UserId: userID(c), IsAdmin: g.IsAdmin(c)})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": musicListJSON(resp.Items)})
}

func (g *Gateway) MusicDelete(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if _, err := g.mus.Delete(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Трек удалён"})
}

func (g *Gateway) MusicPending(c *gin.Context) {
	resp, err := g.mus.ListPending(ctx(), &pbmusic.ListRequest{UserId: userID(c), IsAdmin: true})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": musicListJSON(resp.Items)})
}

func (g *Gateway) MusicApprove(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if _, err := g.mus.Approve(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Трек одобрен"})
}

func (g *Gateway) MusicReject(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID"})
		return
	}
	if _, err := g.mus.Reject(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Трек отклонён"})
}

func musicListJSON(items []*pbmusic.Music) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, m := range items {
		out = append(out, gin.H{
			"id":            m.Id,
			"title":         m.Title,
			"artist":        m.Artist,
			"original_name": m.OriginalName,
			"size":          m.Size,
			"mime_type":     m.MimeType,
			"uploaded_by":   m.UploadedBy,
			"status":        m.Status,
			"created_at":    ts(m.CreatedAt),
		})
	}
	return out
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
