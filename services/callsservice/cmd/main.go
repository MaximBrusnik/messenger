package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"messengermax/callsservice/internal/entity"
	handlergrpc "messengermax/callsservice/internal/handler/grpc"
	"messengermax/callsservice/internal/handler/ws"
	"messengermax/callsservice/internal/repo"
	"messengermax/callsservice/internal/service"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/jwt"
	"messengermax/pkg/nats"
	"messengermax/pkg/postgres"
	pbcalls "messengermax/proto/gen/calls"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "callsservice"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("calls: ", err)
	}
	if err := db.AutoMigrate(&entity.Call{}); err != nil {
		log.Fatal("calls: migrate: ", err)
	}

	callRepo := repo.NewCallRepository(db)
	jwtManager := jwt.NewManager(cfg.JWTSecret)
	hubInstance := ws.New()
	producer := nats.NewProducer(cfg.NATS.URL)
	defer producer.Close()
	srv := service.NewServer(callRepo, producer)
	wsCtrl := ws.NewController(jwtManager, hubInstance, srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sweepMissedCalls(ctx, srv, wsCtrl)

	// gRPC API
	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
			pbcalls.RegisterCallServiceServer(s, handlergrpc.NewServer(srv))
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

func sweepMissedCalls(ctx context.Context, srv *service.Server, ctrl *ws.Controller) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := srv.FindExpiredRinging(time.Now().UnixMilli())
			if err != nil {
				continue
			}
			for i := range expired {
				call, err := srv.ExpireRingingCall(ctx, &expired[i])
				if err != nil {
					continue
				}
				log.Printf("calls: ringing call %d missed (timeout)", call.ID)
				ctrl.NotifyCallEnded(call)
			}
		}
	}
}
