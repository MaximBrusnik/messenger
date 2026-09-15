package main

import (
	"log"

	"google.golang.org/grpc"

	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/postgres"
	sharedredis "messengermax/pkg/redis"
	pb "messengermax/proto/gen/user"

	"messengermax/user/internal/entity"
	"messengermax/user/internal/repo"
	"messengermax/user/internal/service"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "user"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("user: ", err)
	}
	if err := db.AutoMigrate(&entity.Profile{}, &entity.Contact{}); err != nil {
		log.Fatal("user: migrate: ", err)
	}

	profileRepo := repo.NewProfileRepository(db)
	redisClient := sharedredis.New(cfg.Redis.Addr, cfg.Redis.Pass)
	server := service.NewServer(profileRepo, redisClient)

	if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
		pb.RegisterUserServiceServer(s, server)
	}); err != nil {
		log.Fatal("user: ", err)
	}
}
