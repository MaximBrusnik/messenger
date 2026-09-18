package rest

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	reEdge    = regexp.MustCompile(`(?i)(?:EdgA?|EdgiOS)/(\d+)`)
	reOpera   = regexp.MustCompile(`(?i)(?:OPR|Opera)/(\d+)`)
	reSamsung = regexp.MustCompile(`(?i)SamsungBrowser/(\d+)`)
	reChrome  = regexp.MustCompile(`(?i)(?:Chrome|CriOS)/(\d+)`)
	reFirefox = regexp.MustCompile(`(?i)(?:Firefox|FxiOS)/(\d+)`)
	reSafari  = regexp.MustCompile(`(?i)Version/(\d+)`)
	reIE      = regexp.MustCompile(`(?i)(?:MSIE|rv:)(\d+)`)
)

type deviceMeta struct {
	name        string
	platform    string
	fingerprint string
}

func classifyBrowser(ua string) (string, string) {
	if m := reEdge.FindStringSubmatch(ua); m != nil {
		return "Edge " + m[1], "edge"
	}
	if m := reOpera.FindStringSubmatch(ua); m != nil {
		return "Opera " + m[1], "opera"
	}
	if m := reSamsung.FindStringSubmatch(ua); m != nil {
		return "Samsung Internet " + m[1], "samsung"
	}
	if m := reChrome.FindStringSubmatch(ua); m != nil {
		return "Chrome " + m[1], "chrome"
	}
	if m := reFirefox.FindStringSubmatch(ua); m != nil {
		return "Firefox " + m[1], "firefox"
	}
	if strings.Contains(ua, "Safari") {
		if m := reSafari.FindStringSubmatch(ua); m != nil {
			return "Safari " + m[1], "safari"
		}
		return "Safari", "safari"
	}
	if m := reIE.FindStringSubmatch(ua); m != nil {
		return "Internet Explorer " + m[1], "ie"
	}
	return "Браузер", "browser"
}

func classifyPlatform(ua string) string {
	switch {
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"):
		return "iOS"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return "Неизвестная ОС"
	}
}

func metaFromRequest(c *gin.Context) (name, platform, fingerprint, ip string) {
	ua := c.Request.UserAgent()
	name, browser := classifyBrowser(ua)
	platform = classifyPlatform(ua)
	fingerprint = strings.ToLower(browser + "|" + platform)
	ip = c.ClientIP()
	return
}
