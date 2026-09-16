package rest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	pbpush "messengermax/proto/gen/push"
)

// RegisterDevice proxies to pushservice.
func (g *Gateway) RegisterDevice(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	if req.Platform != "android" && req.Platform != "ios" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРёР·РІРµСЃС‚РЅР°СЏ РїР»Р°С‚С„РѕСЂРјР°"})
		return
	}
	_, err := g.psh.RegisterDevice(ctx(), &pbpush.RegisterDeviceRequest{
		UserId: userID(c), Token: req.Token, Platform: req.Platform,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЈСЃС‚СЂРѕР№СЃС‚РІРѕ Р·Р°СЂРµРіРёСЃС‚СЂРёСЂРѕРІР°РЅРѕ"})
}

// UnregisterDevice proxies to pushservice.
func (g *Gateway) UnregisterDevice(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Рµ РґР°РЅРЅС‹Рµ", "details": err.Error()})
		return
	}
	_, err := g.psh.UnregisterDevice(ctx(), &pbpush.UnregisterDeviceRequest{Token: req.Token})
	if err != nil {
		HTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "РЈСЃС‚СЂРѕР№СЃС‚РІРѕ РѕС‚СЂРµРіРёСЃС‚СЂРёСЂРѕРІР°РЅРѕ"})
}

// WSProxy forwards the WebSocket connection to realtimeservice, which
// validates the JWT itself.
func (g *Gateway) WSProxy(c *gin.Context) {
	target, err := url.Parse("http://" + g.cfg.Services.RealtimeWS)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "realtimeservice РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = target.Host
		r.Host = target.Host
		if r.Header.Get("Upgrade") == "websocket" {
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
		}
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

// CallWSProxy forwards the WebSocket connection to call-service, which
// validates the JWT and relays WebRTC signaling between peers.
func (g *Gateway) CallWSProxy(c *gin.Context) {
	target, err := url.Parse("http://" + g.cfg.Services.CallsWS)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "callsservice РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = target.Host
		r.URL.Path = "/ws"
		r.Host = target.Host
		if r.Header.Get("Upgrade") == "websocket" {
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
		}
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

// fileType matches an extension to the legacy upload `type`.
func fileType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "image"
	case ".mp4", ".webm", ".mov":
		return "video"
	case ".mp3", ".wav", ".ogg":
		return "audio"
	default:
		return "file"
	}
}

// Upload stores the attachment locally and returns the public URL.
func (g *Gateway) Upload(c *gin.Context) {
	dir := g.cfg.FileStorageDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "РЅРµ СѓРґР°Р»РѕСЃСЊ РёРЅРёС†РёР°Р»РёР·РёСЂРѕРІР°С‚СЊ С…СЂР°РЅРёР»РёС‰Рµ"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "С„Р°Р№Р» РЅРµ РЅР°Р№РґРµРЅ"})
		return
	}
	defer file.Close()

	name := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	dst := filepath.Join(dir, name)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "РЅРµ СѓРґР°Р»РѕСЃСЊ СЃРѕС…СЂР°РЅРёС‚СЊ С„Р°Р№Р»"})
		return
	}
	size, err := io.Copy(out, file)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "РЅРµ СѓРґР°Р»РѕСЃСЊ СЃРѕС…СЂР°РЅРёС‚СЊ С„Р°Р№Р»"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":  "/uploads/" + name,
		"name": header.Filename,
		"size": size,
		"type": fileType(filepath.Ext(header.Filename)),
		"mime": header.Header.Get("Content-Type"),
	})
}
