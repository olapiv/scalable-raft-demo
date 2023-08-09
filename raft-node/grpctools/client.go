package grpctools

import (
	"context"
	"fmt"
	"time"

	"github.com/olapiv/scalable-raft-demo/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func CreateGrpcConn(host string) (*grpc.ClientConn, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// If we don't use WithBlock(), this can return a "CONNECTING" connection
	// which could potentially fail when actually being used.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", host, conf.GRPC_LISTEN_PORT)
	// Set up a connection to the server
	conn, err := grpc.DialContext(
		ctx,
		addr,
		dialOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC connection to address '%s'; reason: %w", addr, err)
	}
	return conn, nil
}
