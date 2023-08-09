package grpctools

import (
	"fmt"
	"net"
	"time"

	"github.com/olapiv/scalable-raft-demo/conf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func NewTcpListener(log *zap.Logger) (lis net.Listener, cleanup func(), err error) {
	grpcAddress := conf.GetServerGRPCAddress("")
	lis, err = net.Listen("tcp", grpcAddress)
	if err != nil {
		return nil, cleanup, fmt.Errorf("cannot listen to TCP; reason: %w", err)
	}
	cleanup = func() {
		log.Info(fmt.Sprintf("Closing TCP listener at '%s'", lis.Addr()))
		err = lis.Close()
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to close TCP listener; reason: %v", err))
		}
	}
	return lis, cleanup, nil
}

func NewServer(log *zap.Logger) (s *grpc.Server, cleanup func()) {
	s = grpc.NewServer()

	cleanup = func() {
		log.Info("Trying to stop gRPC server gracefully")

		stopped := make(chan bool, 1)
		go func() {
			// This will take forever if e.g. the leader attempts continue heartbeating
			s.GracefulStop()
			close(stopped)
		}()

		t := time.NewTimer(3 * time.Second)
		select {
		case <-t.C:
			// This is very likely to happen if we are still receiving heartbeats
			// from the Raft leader
			s.Stop()
			log.Info("Forcefully stopped gRPC server due to timeout")
		case <-stopped:
			t.Stop()
			log.Info("Successfully stopped gRPC server gracefully")
		}
	}
	return s, cleanup
}
