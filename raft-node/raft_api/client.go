package raft_api

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/raft"
	"github.com/olapiv/scalable-raft-demo/grpctools"
	"github.com/olapiv/scalable-raft-demo/raft_api/raft_api_proto"
	"go.uber.org/zap"
)

// Find leader and then ask leader
func RequestJoinCluster(
	log *zap.Logger,
	startingHost string,
	selfId raft.ServerID,
	selfAddress string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	partOfCluster, err := isPartOfCluster(ctx, log, startingHost)
	if err != nil {
		return err
	} else if !partOfCluster {
		return fmt.Errorf("the host %s is not part of the Raft cluster", startingHost)
	}
	log.Sugar().Infof("The host %s is part of the Raft cluster, we can ask it who the leader is.", startingHost)

	// TODO: Consider doing this with retries in case the leader changes inbetween
	var leaderHost string
	for {
		reply, err := getLeader(ctx, log, startingHost)
		if err != nil {
			return err
		}

		if reply == nil {
			return fmt.Errorf("received an empty reply when asking '%s' for the leader address", startingHost)
		}

		log.Sugar().Infof("Received reply that id '%s' with address '%s' is the leader of the Raft cluster",
			reply.ServerID,
			reply.ServerAddress,
		)

		if reply.ServerID == "" {
			log.Info("Looks like we're in an election, let's try again in a bit")
			time.Sleep(1 * time.Second)
			continue
		}

		host, _, err := net.SplitHostPort(reply.ServerAddress)
		if err != nil {
			return err
		}

		leaderHost = host
		break
	}

	return requestJoinCluster(
		ctx,
		log,
		leaderHost,
		selfId,
		selfAddress,
	)
}

// Make a call to the leader
func requestJoinCluster(
	ctx context.Context,
	log *zap.Logger,
	leaderHost string,
	selfId raft.ServerID,
	selfAddress string,
) error {
	log.Sugar().Infof("Requesting leader on host '%s' to join the Raft cluster with id '%s'",
		leaderHost,
		selfId,
	)

	conn, err := grpctools.CreateGrpcConn(leaderHost)
	if err != nil {
		return err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Error(err.Error())
		}
	}()

	raftApiClient := raft_api_proto.NewRaftApiClient(conn)

	reply, err := raftApiClient.AddVoter(
		ctx,
		&raft_api_proto.AddVoterRequest{
			ServerID:      string(selfId),
			ServerAddress: selfAddress,
			PrevIndex:     0,
		},
	)
	if err != nil {
		return err
	}
	if reply != nil {
		log.Sugar().Infof("Message from host '%s': %s", leaderHost, reply.Message)
	}
	return nil
}

func RequestLeaveCluster(
	log *zap.Logger,
	raftObject *raft.Raft,
	selfId raft.ServerID,
	fullyRemove bool,
) {
	if fullyRemove {
		log.Warn("Trying to leave the Raft cluster fully now")
	} else {
		log.Warn("Trying to leave the Raft cluster temporarily now")
	}
	/*
		Taking time for this is important; the cluster will crash
		sooner or later if some nodes have not disconnected
		properly because the leader will be forced to regularly
		run a leader election.

		TODO: Add some logic whereby the leader will remove unreachable
			nodes from the cluster.
	*/
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			log.Error("Timed out attempting to leave the Raft cluster")
			return
		case <-time.After(1 * time.Second):
			leaderAddress, leaderId := raftObject.LeaderWithID()

			// In case of leader election
			if leaderId == "" {
				log.Info("Current leader id is empty, will try again")
				continue
			}

			log.Sugar().Warnf("Requesting leader id '%s' with address '%s' to remove self (Raft id '%s') from Raft cluster",
				leaderId,
				leaderAddress,
				selfId,
			)

			// Perhaps error due to leader election
			leaderHost, _, err := net.SplitHostPort(string(leaderAddress))
			if err != nil {
				log.Error(err.Error())
				return // Odd error to run into; give up
			}

			err = requestLeaveCluster(
				log,
				leaderHost,
				selfId,
				fullyRemove,
			)
			if err != nil {
				// Could be that leader also wants to exit
				log.Sugar().Errorf("Failed requesting leader id '%s' to remove self (Raft id '%s') from Raft cluster; reason: %v",
					leaderId,
					selfId,
					err,
				)
				continue // Try again, leader may have changed inbetween
			}
			return
		}
	}
}

// Make call to leader
func requestLeaveCluster(
	log *zap.Logger,
	leaderHost string,
	selfId raft.ServerID,
	fullyRemove bool,
) error {
	log.Sugar().Infof("Requesting host '%s' to remove server with id '%s' from Raft cluster",
		leaderHost,
		selfId,
	)

	conn, err := grpctools.CreateGrpcConn(leaderHost)
	if err != nil {
		return err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Error(err.Error())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	raftApiClient := raft_api_proto.NewRaftApiClient(conn)

	var reply *raft_api_proto.GenericReply
	if fullyRemove {
		reply, err = raftApiClient.RemoveServer(
			ctx,
			&raft_api_proto.RemoveServerRequest{
				ServerID:  string(selfId),
				PrevIndex: 0,
			},
		)
	} else {
		/*
			This will cause us to be deactivated; when starting up again
			we will read the cluster configuration, see that we were
			inactive and not expect to receive heartbeats straight
			away again.
		*/
		reply, err = raftApiClient.DemoteVoter(
			ctx,
			&raft_api_proto.RemoveServerRequest{
				ServerID:  string(selfId),
				PrevIndex: 0,
			},
		)
	}
	if err != nil {
		return err
	}
	if reply != nil {
		log.Sugar().Infof("Message from host '%s': %s", leaderHost, reply.Message)
	}
	return nil
}

func isPartOfCluster(
	ctx context.Context,
	log *zap.Logger,
	queryHost string,
) (bool, error) {
	log.Sugar().Infof("Querying host '%s' whether it is part of the Raft cluster", queryHost)

	conn, err := grpctools.CreateGrpcConn(queryHost)
	if err != nil {
		return false, err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Error(err.Error())
		}
	}()

	raftApiClient := raft_api_proto.NewRaftApiClient(conn)

	reply, err := raftApiClient.IsPartOfCluster(ctx, &raft_api_proto.Empty{})
	if err != nil {
		return false, err
	} else if reply == nil {
		return false, fmt.Errorf("the reply body is empty for checking whether host '%s' is part of the Raft cluster", queryHost)
	}
	return reply.PartOfCluster, nil
}

func getLeader(
	ctx context.Context,
	log *zap.Logger,
	queryHost string,
) (*raft_api_proto.LeaderWithIDReply, error) {
	log.Sugar().Infof("Querying host '%s' for the leader of the Raft cluster", queryHost)

	conn, err := grpctools.CreateGrpcConn(queryHost)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Error(err.Error())
		}
	}()

	raftApiClient := raft_api_proto.NewRaftApiClient(conn)

	return raftApiClient.LeaderWithID(ctx, &raft_api_proto.Empty{})
}
