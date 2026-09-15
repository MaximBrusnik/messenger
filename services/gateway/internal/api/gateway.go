package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	pbauth "messengermax/proto/gen/auth"
	pbcalls "messengermax/proto/gen/calls"
	pbchat "messengermax/proto/gen/chat"
	pbmusic "messengermax/proto/gen/music"
	pbpush "messengermax/proto/gen/push"
	pbuser "messengermax/proto/gen/user"
)

type Gateway struct {
	cfg   *config.Config
	auth  pbauth.AuthServiceClient
	usr   pbuser.UserServiceClient
	chat  pbchat.ChatServiceClient
	psh   pbpush.PushServiceClient
	mus   pbmusic.MusicServiceClient
	calls pbcalls.CallServiceClient
}

func NewGateway(cfg *config.Config) (*Gateway, error) {
	g := &Gateway{cfg: cfg}

	var conns []*grpc.ClientConn
	closeAll := func() {
		for _, cc := range conns {
			_ = cc.Close()
		}
	}

	if cc, err := grpcsrv.Dial(cfg.Services.AuthAddr); err != nil {
		closeAll()
		return nil, err
	} else {
		conns = append(conns, cc)
		g.auth = pbauth.NewAuthServiceClient(cc)
	}

	if cc, err := grpcsrv.Dial(cfg.Services.UserAddr); err != nil {
		closeAll()
		return nil, err
	} else {
		conns = append(conns, cc)
		g.usr = pbuser.NewUserServiceClient(cc)
	}

	if cc, err := grpcsrv.Dial(cfg.Services.ChatAddr); err != nil {
		closeAll()
		return nil, err
	} else {
		conns = append(conns, cc)
		g.chat = pbchat.NewChatServiceClient(cc)
	}

	cc, err := grpcsrv.Dial(cfg.Services.MusicAddr)
	if err != nil {
		closeAll()
		return nil, err
	}
	conns = append(conns, cc)
	g.mus = pbmusic.NewMusicServiceClient(cc)

	cc2, err := grpcsrv.Dial(cfg.Services.RealtimeAddr)
	if err != nil {
		closeAll()
		return nil, err
	}
	conns = append(conns, cc2)

	cc3, err := grpcsrv.Dial(cfg.Services.PushAddr)
	if err != nil {
		closeAll()
		return nil, err
	}
	conns = append(conns, cc3)
	g.psh = pbpush.NewPushServiceClient(cc3)

	// Calls are best-effort; if the service is unavailable we degrade
	// gracefully by not proxying call endpoints rather than refusing to boot.
	if cc4, err := grpcsrv.Dial(cfg.Services.CallsAddr); err == nil {
		conns = append(conns, cc4)
		g.calls = pbcalls.NewCallServiceClient(cc4)
	} else {
		log.Printf("gateway: calls-service unreachable at %s: %v", cfg.Services.CallsAddr, err)
	}

	return g, nil
}

func HTTPError(c *gin.Context, err error) {
	st, _ := status.FromError(err)
	msg := st.Message()
	if msg == "" {
		msg = "внутренняя ошибка"
	}
	var code int
	switch st.Code() {
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.PermissionDenied:
		code = http.StatusForbidden
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.AlreadyExists:
		code = http.StatusConflict
	case codes.ResourceExhausted:
		code = http.StatusRequestEntityTooLarge
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	default:
		code = http.StatusInternalServerError
	}
	c.JSON(code, gin.H{"error": msg})
}

func userID(c *gin.Context) uint64 {
	v, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	if id, ok := v.(uint); ok {
		return uint64(id)
	}
	return 0
}

func parseID(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(c.Param(name), 10, 64)
}

func ctx() context.Context { return context.Background() }

func ts(t *timestamppb.Timestamp) string {
	if t == nil || t.Seconds == 0 {
		return ""
	}
	return t.AsTime().Format(time.RFC3339)
}
