package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"

	"messengermax/authservice/internal/entity"
	handlergrpc "messengermax/authservice/internal/handler/grpc"
	"messengermax/authservice/internal/integration/email"
	"messengermax/authservice/internal/repo"
	"messengermax/authservice/internal/service"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/jwt"
	"messengermax/pkg/postgres"
	pbauth "messengermax/proto/gen/auth"
	pbuser "messengermax/proto/gen/user"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "authservice"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("auth: ", err)
	}
	if err := db.AutoMigrate(&entity.User{}, &entity.Session{}); err != nil {
		log.Fatal("auth: migrate: ", err)
	}

	userRepo := repo.NewUserRepository(db)
	jwtManager := jwt.NewManager(cfg.JWTSecret)
	emailSvc := email.NewService(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User, cfg.SMTP.Pass, cfg.SMTP.From, cfg.App.AppURL)

	var userClient pbuser.UserServiceClient
	userConn, err := dialWithRetry(context.Background(), cfg.Services.UserAddr, 30*time.Second, 2*time.Second)
	if err != nil {
		log.Printf("auth: userservice unreachable at %s even after retries: %v", cfg.Services.UserAddr, err)
	} else {
		defer userConn.Close()
		userClient = pbuser.NewUserServiceClient(userConn)
	}

	server := service.NewServer(userRepo, jwtManager, emailSvc, cfg.App.RequireEmailVerification, cfg.App.MaxUsersEnabled, cfg.App.MaxUsersLimit, userClient)

	if err := seedDefaults(userRepo, userClient); err != nil {
		log.Printf("auth: seed defaults: %v", err)
	}

	if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
		pbauth.RegisterAuthServiceServer(s, handlergrpc.NewServer(server))
	}); err != nil {
		log.Fatal("auth: ", err)
	}
}

func dialWithRetry(ctx context.Context, addr string, totalTimeout, interval time.Duration) (*grpc.ClientConn, error) {
	deadline := time.Now().Add(totalTimeout)
	for {
		conn, err := grpcsrv.Dial(addr)
		if err == nil {
			return conn, nil
		}
		if time.Now().Add(interval).After(deadline) {
			return nil, err
		}
		log.Printf("auth: userservice at %s not ready: %v; retrying in %s", addr, err, interval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}
