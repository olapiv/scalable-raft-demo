package raft_api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/raft"
	"github.com/olapiv/scalable-raft-demo/raft_api/raft_api_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *RaftApiServer) AddVoter(
	ctx context.Context,
	in *raft_api_proto.AddVoterRequest,
) (*raft_api_proto.GenericReply, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "argument AddVoterRequest cannot be nil")
	}

	s.log.Sugar().Infof("Received call to add voter id '%s' with address '%s' to Raft cluster",
		in.ServerID,
		in.ServerAddress,
	)

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(20 * time.Second)
	}

	if err := s.raft.AddVoter(
		raft.ServerID(in.ServerID),
		raft.ServerAddress(in.ServerAddress),
		in.PrevIndex,
		time.Until(deadline),
	).Error(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add voter to Raft cluster; reason: %v", err)
	}

	msg := fmt.Sprintf("Successfully added voter id '%s' with address '%s' to Raft cluster",
		in.ServerID,
		in.ServerAddress,
	)
	s.log.Info(msg)
	return &raft_api_proto.GenericReply{Message: msg}, nil
}

func (s *RaftApiServer) LeaderWithID(
	ctx context.Context,
	in *raft_api_proto.Empty,
) (*raft_api_proto.LeaderWithIDReply, error) {
	serverAddress, serverId := s.raft.LeaderWithID()
	return &raft_api_proto.LeaderWithIDReply{
		ServerID:      string(serverId),
		ServerAddress: string(serverAddress),
	}, nil
}

func (s *RaftApiServer) RemoveServer(
	ctx context.Context,
	in *raft_api_proto.RemoveServerRequest,
) (*raft_api_proto.GenericReply, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "argument RemoveServerRequest cannot be nil")
	}

	s.log.Sugar().Infof("Received call to remove server id '%s' from Raft cluster", in.ServerID)

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(20 * time.Second)
	}

	if err := s.raft.RemoveServer(
		raft.ServerID(in.ServerID),
		in.PrevIndex,
		time.Until(deadline),
	).Error(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove server id '%s' from Raft cluster; reason: %v", in.ServerID, err)
	}

	msg := fmt.Sprintf("Successfully removed voter id '%s' from Raft cluster", in.ServerID)
	s.log.Info(msg)

	return &raft_api_proto.GenericReply{Message: msg}, nil
}

func (s *RaftApiServer) DemoteVoter(ctx context.Context, in *raft_api_proto.RemoveServerRequest) (*raft_api_proto.GenericReply, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "argument RemoveServerRequest cannot be nil")
	}

	s.log.Sugar().Infof("Received call to demote voter id '%s' in Raft cluster", in.ServerID)

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(20 * time.Second)
	}

	if err := s.raft.DemoteVoter(
		raft.ServerID(in.ServerID),
		in.PrevIndex,
		time.Until(deadline),
	).Error(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to demote voter in Raft cluster; reason: %v", err)
	}

	msg := fmt.Sprintf("Successfully demoted voter id '%s' in Raft cluster", in.ServerID)
	s.log.Info(msg)

	return &raft_api_proto.GenericReply{Message: msg}, nil
}

func (s *RaftApiServer) IsPartOfCluster(
	ctx context.Context,
	in *raft_api_proto.Empty,
) (*raft_api_proto.IsPartOfClusterReply, error) {

	s.log.Debug("Received call to ask whether we are part of the Raft cluster")

	raftState := s.raft.State()
	if raftState == raft.Leader || raftState == raft.Candidate {
		return &raft_api_proto.IsPartOfClusterReply{
				PartOfCluster: true,
			},
			nil
	}

	if val, ok := s.raft.Stats()["num_peers"]; !ok {
		s.log.Sugar().Warnf("The key num_peers does not exist in Stats; Stats: %v", s.raft.Stats())
		return &raft_api_proto.IsPartOfClusterReply{
				PartOfCluster: false,
			},
			nil
	} else {
		numPeers, err := strconv.Atoi(val)
		if err != nil {
			s.log.Error(err.Error())
			return nil, status.Error(codes.Internal, err.Error())
		}
		s.log.Sugar().Debugf("Our Raft state: %s, Number of peers: %d", raftState, numPeers)
		partOfCluster := false
		if numPeers > 0 {
			partOfCluster = true
		}
		return &raft_api_proto.IsPartOfClusterReply{
				PartOfCluster: partOfCluster,
			},
			nil
	}
}
