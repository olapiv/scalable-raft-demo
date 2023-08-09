package myraft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	transport "github.com/Jille/raft-grpc-transport"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/olapiv/scalable-raft-demo/conf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewRaft(
	log *zap.Logger,
	isBootstrap bool,
	myHostname string,
	fsm raft.FSM,
) (*raft.Raft, raft.ServerID, *transport.Manager, error) {
	// Assigning the hostname to the raft ID simplifies reading logs
	myID := raft.ServerID(myHostname)

	log.Sugar().Infof("Instantiating Raft using node id '%s'; Bootstrap cluster: %v", myID, isBootstrap)

	c := raft.DefaultConfig()
	c.LocalID = myID

	c.Logger = hclog.FromStandardLogger(zap.NewStdLog(log), &hclog.LoggerOptions{
		Name: "RAFT",
	})

	raftDir := "/root/raft"
	baseDir := filepath.Join(raftDir, string(myID))
	err := os.MkdirAll(baseDir, 0777)
	if err != nil {
		return nil, myID, nil, err
	}

	logsDB, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "logs.dat"))
	if err != nil {
		return nil, myID, nil, err
	}

	stableDB, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "stable.dat"))
	if err != nil {
		return nil, myID, nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(baseDir, 3, os.Stderr)
	if err != nil {
		return nil, myID, nil, fmt.Errorf(`raft.NewFileSnapshotStore(%q, ...): %v`, baseDir, err)
	}

	// Important to use the full address here, since Raft otherwise does not know it
	serverAddress := raft.ServerAddress(conf.GetServerGRPCAddress(myHostname))
	log.Sugar().Infof("Using server address '%s' for Raft", serverAddress)

	tm := transport.New(serverAddress, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})

	r, err := raft.NewRaft(c, fsm, logsDB, stableDB, snapshotStore, tm.Transport())
	if err != nil {
		return nil, myID, nil, fmt.Errorf("failed creating new Raft object; reason: %w", err)
	}

	if isBootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(myID),
					Address:  serverAddress,
				},
			},
		}
		f := r.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			return nil, myID, nil, fmt.Errorf("failed bootstrapping the Raft cluster: %w", err)
		}

		// Make sure leader election has happened before using the Raft instantation (and writing to state)
		raftConf := r.ReloadableConfig()
		time.Sleep(raftConf.ElectionTimeout + raftConf.HeartbeatTimeout)
	}

	log.Sugar().Infof("Successfully instantiated Raft cluster; Last Raft index: %d; Applied Raft index: %d",
		r.LastIndex(),
		r.AppliedIndex(),
	)

	return r, myID, tm, nil
}

// Call to send payload to consensus algorithm
func MarshallAndApply(raftObj *raft.Raft, desiredState *DesiredState) error {
	dataM, err := json.Marshal(desiredState)
	if err != nil {
		return err
	}
	return raftObj.Apply(dataM, time.Second).Error()
}
