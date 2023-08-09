package raft_api

import (
	"github.com/hashicorp/raft"
	"github.com/olapiv/scalable-raft-demo/raft_api/raft_api_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RaftApiServer struct {
	raft_api_proto.UnimplementedRaftApiServer
	log  *zap.Logger
	raft *raft.Raft
}

func newRaftApiServer(log *zap.Logger, raft *raft.Raft) *RaftApiServer {
	return &RaftApiServer{
		log:  log,
		raft: raft,
	}
}

func RegisterGrpcFuncs(log *zap.Logger, s *grpc.Server, raft *raft.Raft) {
	raftApiServer := newRaftApiServer(log, raft)
	raft_api_proto.RegisterRaftApiServer(s, raftApiServer)
}
