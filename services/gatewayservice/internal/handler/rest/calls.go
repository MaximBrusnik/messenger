package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pbcalls "messengermax/proto/gen/calls"
)

// GetCallHistory returns the paginated call history for the authenticated user.
func (g *Gateway) GetCallHistory(c *gin.Context) {
	if g.calls == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "СЃРµСЂРІРёСЃ Р·РІРѕРЅРєРѕРІ РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	uid := userID(c)
	resp, err := g.calls.GetCallHistory(c.Request.Context(), &pbcalls.GetCallHistoryRequest{
		UserId: uid,
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		HTTPError(c, err)
		return
	}
	out := make([]gin.H, 0, len(resp.Calls))
	for _, cs := range resp.Calls {
		out = append(out, gin.H{
			"id":             cs.Id,
			"caller_id":      cs.CallerId,
			"callee_id":      cs.CalleeId,
			"call_type":      callTypeString(cs.CallType),
			"status":         callStatusString(cs.Status),
			"started_at_ms":  cs.StartedAtMs,
			"accepted_at_ms": cs.AcceptedAtMs,
			"ended_at_ms":    cs.EndedAtMs,
			"duration_ms":    cs.DurationMs,
			"end_reason":     cs.EndReason,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// GetCall returns the details of a single call.
func (g *Gateway) GetCall(c *gin.Context) {
	if g.calls == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "СЃРµСЂРІРёСЃ Р·РІРѕРЅРєРѕРІ РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	id, err := parseID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "РЅРµРІРµСЂРЅС‹Р№ ID Р·РІРѕРЅРєР°"})
		return
	}
	// FindActiveCall does not support single-call lookup; return the session
	// data when it's still active. For history calls we need a dedicated
	// GetCall RPC; use GetActiveCall for now.
	uid := userID(c)
	resp, err := g.calls.GetActiveCall(c.Request.Context(), &pbcalls.GetActiveCallRequest{UserId: uid})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Р·РІРѕРЅРѕРє РЅРµ РЅР°Р№РґРµРЅ"})
		return
	}
	if resp.Id != id {
		c.JSON(http.StatusNotFound, gin.H{"error": "Р·РІРѕРЅРѕРє РЅРµ РЅР°Р№РґРµРЅ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":             resp.Id,
		"caller_id":      resp.CallerId,
		"callee_id":      resp.CalleeId,
		"call_type":      callTypeString(resp.CallType),
		"status":         callStatusString(resp.Status),
		"started_at_ms":  resp.StartedAtMs,
		"accepted_at_ms": resp.AcceptedAtMs,
		"ended_at_ms":    resp.EndedAtMs,
		"duration_ms":    resp.DurationMs,
		"end_reason":     resp.EndReason,
	}})
}

// GetActiveCall returns the currently active call for the authenticated user, if any.
func (g *Gateway) GetActiveCall(c *gin.Context) {
	if g.calls == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "СЃРµСЂРІРёСЃ Р·РІРѕРЅРєРѕРІ РЅРµРґРѕСЃС‚СѓРїРµРЅ"})
		return
	}
	resp, err := g.calls.GetActiveCall(c.Request.Context(), &pbcalls.GetActiveCallRequest{UserId: userID(c)})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":             resp.Id,
		"caller_id":      resp.CallerId,
		"callee_id":      resp.CalleeId,
		"call_type":      callTypeString(resp.CallType),
		"status":         callStatusString(resp.Status),
		"started_at_ms":  resp.StartedAtMs,
		"accepted_at_ms": resp.AcceptedAtMs,
		"ended_at_ms":    resp.EndedAtMs,
		"duration_ms":    resp.DurationMs,
		"end_reason":     resp.EndReason,
	}})
}

// GetCallConfig returns the ICE server configuration used for WebRTC peer
// connections, so clients never hardcode STUN/TURN endpoints.
func (g *Gateway) GetCallConfig(c *gin.Context) {
	servers := make([]gin.H, 0, len(g.cfg.Calls.StunServers)+len(g.cfg.Calls.TurnServers))
	for _, s := range g.cfg.Calls.StunServers {
		servers = append(servers, gin.H{"urls": s})
	}
	for _, t := range g.cfg.Calls.TurnServers {
		servers = append(servers, gin.H{
			"urls":       t.URLs,
			"username":   t.Username,
			"credential": t.Credential,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ice_servers": servers}})
}

func callTypeString(ct pbcalls.CallType) string {
	switch ct {
	case pbcalls.CallType_CALL_TYPE_VIDEO:
		return "video"
	default:
		return "audio"
	}
}

func callStatusString(cs pbcalls.CallStatus) string {
	switch cs {
	case pbcalls.CallStatus_CALL_STATUS_RINGING:
		return "ringing"
	case pbcalls.CallStatus_CALL_STATUS_ACTIVE:
		return "active"
	case pbcalls.CallStatus_CALL_STATUS_ENDED:
		return "ended"
	case pbcalls.CallStatus_CALL_STATUS_MISSED:
		return "missed"
	case pbcalls.CallStatus_CALL_STATUS_REJECTED:
		return "rejected"
	case pbcalls.CallStatus_CALL_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}
