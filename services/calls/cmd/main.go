package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"messengermax/calls/internal/entity"
	"messengermax/calls/internal/hub"
	"messengermax/calls/internal/repo"
	"messengermax/calls/internal/service"
	"messengermax/calls/internal/ws"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/jwt"
	"messengermax/pkg/postgres"
	pbcalls "messengermax/proto/gen/calls"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "calls"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("calls: ", err)
	}
	if err := db.AutoMigrate(&entity.Call{}); err != nil {
		log.Fatal("calls: migrate: ", err)
	}

	callRepo := repo.NewCallRepository(db)
	jwtManager := jwt.NewManager(cfg.JWTSecret)
	hubInstance := hub.New()
	srv := service.NewServer(callRepo)
	wsCtrl := ws.NewController(jwtManager, hubInstance, srv)

	// expire ringing calls that nobody accepted
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sweepMissedCalls(ctx, callRepo, wsCtrl)

	// gRPC API
	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
			pbcalls.RegisterCallServiceServer(s, srv)
		})
	}()

	// HTTP signaling endpoint
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.GET("/ws", wsCtrl.Handle)
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	httpErr := make(chan error, 1)
	go func() { httpErr <- router.Run(":" + cfg.HTTPPort) }()

	log.Printf("calls: ws on :%s, grpc on :%s", cfg.HTTPPort, cfg.GRPCPort)

	select {
	case err := <-grpcErr:
		log.Fatal("calls: grpc: ", err)
	case err := <-httpErr:
		log.Fatal("calls: http: ", err)
	}
}

// sweepMissedCalls marks ringing calls that exceeded the ring timeout as
// missed, notifying participants over the signaling channel.
func sweepMissedCalls(ctx context.Context, callRepo repo.CallRepository, ctrl *ws.Controller) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := callRepo.FindExpiredRinging(time.Now().UnixMilli())
			if err != nil {
				continue
			}
			for i := range expired {
				expired[i].Status = entity.CallStatusMissed
				expired[i].EndedAtMs = time.Now().UnixMilli()
				expired[i].EndReason = "timeout"
				_ = callRepo.Update(&expired[i])
				log.Printf("calls: ringing call %d missed (timeout)", expired[i].ID)
				ctrl.NotifyCallEnded(&expired[i])
			}
		}
	}
}
