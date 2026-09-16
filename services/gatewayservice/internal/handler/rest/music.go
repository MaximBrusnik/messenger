package rest

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	pbmusic "messengermax/proto/gen/music"
	pbuser "messengermax/proto/gen/user"
)

// MusicFileProxy forwards upload/stream/download to the musicservice,
// injecting the gateway's trusted identity headers.
func (g *Gateway) MusicFileProxy(c *gin.Context) {
	target, err := url.Parse("http://" + g.cfg.MusicHTTPAddr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "musicservice РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	uidHeader := itoa(userID(c))
	adminHeader := boolToStr(g.IsAdmin(c))
	// Strip the /api/v1 prefix: the music service registers its routes at the
	// root (/music/upload, /music/:id/stream, ...), while the public API is
	// exposed under /api/v1/music/*.
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = target.Host
		r.Host = target.Host
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.Header.Set("X-User-Id", uidHeader)
		r.Header.Set("X-Is-Admin", adminHeader)
	}
	proxy.ServeHTTP(c.Writer, c.Request)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID"})
		return
	}
	if _, err := g.mus.Delete(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РўСЂРµРє СѓРґР°Р»С‘РЅ"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID"})
		return
	}
	if _, err := g.mus.Approve(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РўСЂРµРє РѕРґРѕР±СЂРµРЅ"})
}

func (g *Gateway) MusicReject(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID"})
		return
	}
	if _, err := g.mus.Reject(ctx(), &pbmusic.GetMusicRequest{Id: id}); err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РўСЂРµРє РѕС‚РєР»РѕРЅС‘РЅ"})
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
