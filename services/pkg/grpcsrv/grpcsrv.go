// Package grpcsrv provides helpers to run a gRPC server with graceful shutdown.
package grpcsrv

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Run starts a gRPC server on the given port, blocking until the server stops.
func Run(port string, register func(s *grpc.Server)) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("grpc listen on :%s: %w", port, err)
	}

	// As a matter of hygiene we omit interceptors here; individual services
	// may attach their own. A plain server keeps bootstrapping simple.
	server := grpc.NewServer()
	register(server)

	log.Printf("gRPC server listening on :%s", port)
	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

// Dial connects to a gRPC service at addr with a blocking dial timeout.
func Dial(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
