package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"messengermax/musicservice/internal/entity"
	handlergrpc "messengermax/musicservice/internal/handler/grpc"
	"messengermax/musicservice/internal/handler/rest"
	"messengermax/musicservice/internal/integration/filestore"
	"messengermax/musicservice/internal/repo"
	"messengermax/musicservice/internal/service"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/postgres"
	pb "messengermax/proto/gen/music"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "musicservice"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("music: postgres: ", err)
	}
	if err := db.AutoMigrate(&entity.Music{}); err != nil {
		log.Fatal("music: migrate: ", err)
	}

	st, err := filestore.New(cfg.MusicStorageDir)
	if err != nil {
		log.Fatal("music: store: ", err)
	}
	musicRepo := repo.NewMusicRepository(db)

	go func() {
		if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
			pb.RegisterMusicServiceServer(s, handlergrpc.NewServer(service.NewServer(musicRepo, st)))
		}); err != nil {
			log.Fatal("music: grpc: ", err)
		}
	}()
	log.Printf("music: gRPC on :%s", cfg.GRPCPort)

	app := gin.New()
	app.Use(gin.Recovery())
	handler := rest.NewHandler(musicRepo, st)
	// file endpoints are fronted by the api-gateway, which injects identity headers
	app.Use(rest.GatewayIdentity())
	app.POST("/music/upload", handler.Upload)
	app.GET("/music/:id/stream", handler.Stream)
	app.GET("/music/:id/download", handler.Download)

	go func() {
		log.Printf("music: HTTP on :%s", cfg.HTTPPort)
		if err := app.Run(":" + cfg.HTTPPort); err != nil && err != http.ErrServerClosed {
			log.Fatal("music: http: ", err)
		}
	}()

	select {}
}
