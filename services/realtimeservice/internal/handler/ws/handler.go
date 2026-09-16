package ws

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"messengermax/pkg/jwt"
	"messengermax/pkg/nats"
	"messengermax/pkg/redis"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Controller manages connection lifecycle: HTTP upgrade, heartbeat,
// online markers in Redis, status broadcasting and programmatic teardown.
type Controller struct {
	jwtManager *jwt.Manager
	hub        *Hub
	redis      *redis.Client
	producer   *nats.Producer
}

func NewController(jwtManager *jwt.Manager, h *Hub, redis *redis.Client, producer *nats.Producer) *Controller {
	return &Controller{jwtManager: jwtManager, hub: h, redis: redis, producer: producer}
}

// Handle upgrades the request and runs the connection's read loop.
func (c *Controller) Handle(gctx *gin.Context) {
	token := gctx.Query("token")
	if token == "" {
		gctx.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}
	userID, err := c.jwtManager.ValidateToken(token)
	if err != nil {
		gctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(gctx.Writer, gctx.Request, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}

	c.hub.Register(userID, conn)
	hctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = c.redis.SetOnline(hctx, userID)
	c.hub.SendToUser(userID, WSMessage{Type: "CONNECTED", Payload: gin.H{"user_id": userID}})
	c.hub.Broadcast(WSMessage{Type: "USER_STATUS", Payload: gin.H{"user_id": userID, "status": "online"}})
	c.producer.Publish(nats.TopicUserEvents, "conn", nats.EventUserStatus{UserID: int64(userID), Online: true})

	go c.pingLoop(conn)
	c.readLoop(conn, userID, cancel)
}

// DisconnectUser programmatically tears down a user's connection and
// publishes the offline status.
func (c *Controller) DisconnectUser(ctx context.Context, userID uint) error {
	conn := c.hub.Disconnect(userID)
	if conn != nil {
		_ = conn.Close()
	}
	_ = c.redis.SetOffline(ctx, userID)
	c.hub.Broadcast(WSMessage{Type: "USER_STATUS", Payload: gin.H{"user_id": userID, "status": "offline"}})
	c.producer.Publish(nats.TopicUserEvents, "disc", nats.EventUserStatus{UserID: int64(userID), Online: false})
	return nil
}

func (c *Controller) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			_ = conn.Close()
			return
		}
	}
}

func (c *Controller) readLoop(conn *websocket.Conn, userID uint, cancel context.CancelFunc) {
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		hctx, cc := context.WithTimeout(context.Background(), time.Second)
		_ = c.redis.SetOnline(hctx, userID)
		cc()
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	cancel()
	c.hub.Unregister(userID, conn)
	_ = conn.Close()
	// The connection may have been replaced by a newer one (e.g. a refresh or
	// reconnect that registered a new socket for the same user before this
	// cleanup ran). In that case the user is still online — don't clear the
	// Redis marker or broadcast an offline event.
	if c.hub.IsOnline(userID) {
		return
	}
	hctx, cc := context.WithTimeout(context.Background(), time.Second)
	defer cc()
	_ = c.redis.SetOffline(hctx, userID)
	c.hub.Broadcast(WSMessage{Type: "USER_STATUS", Payload: gin.H{"user_id": userID, "status": "offline"}})
	c.producer.Publish(nats.TopicUserEvents, "disc", nats.EventUserStatus{UserID: int64(userID), Online: false})
}
