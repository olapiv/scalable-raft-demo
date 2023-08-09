package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/olapiv/scalable-raft-demo/conf"
	"github.com/olapiv/scalable-raft-demo/grpctools"
	"github.com/olapiv/scalable-raft-demo/http_client"
	"github.com/olapiv/scalable-raft-demo/myraft"
	"github.com/olapiv/scalable-raft-demo/raft_api"
	"github.com/olapiv/scalable-raft-demo/reconciliation"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// We do this so that we can exit with error code 1 without discarding defer() functions
	retcode := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("caught a panic in main(); making sure we are not returning with exit code 0;\nrecovery: %s\nstacktrace:\n%s", r, debug.Stack())
			retcode = 1
		}
		fmt.Printf("Running os.Exit(%d)\n", retcode)
		os.Exit(retcode)
	}()

	logger, err := setupLogger()
	if err != nil {
		fmt.Println(err.Error())
		retcode = 1
		return
	}

	isBootstrapHost, err := strconv.ParseBool((os.Getenv(conf.BOOTSTRAP_HOST_ENV)))
	if err != nil {
		retcode = 1
		logger.Error(err.Error())
		return
	}

	httpClient, err := http_client.New(logger)
	if err != nil {
		logger.Sugar().Errorf("Failed creating a HTTP client reason: %v", err)
		retcode = 1
		return
	}

	time.Sleep(3 * time.Second) // Make sure all containers are registered
	currentHosts, err := httpClient.GetCurrentHosts()
	if err != nil {
		logger.Sugar().Errorf("Failed to GET all participating hosts; reason: %v", err)
		retcode = 1
		return
	}

	logger.Sugar().Infof("The following hosts are participants in the Raft cluster: %v", currentHosts)

	fsm := myraft.NewMyFSM(logger)

	myHostname := os.Getenv(conf.HOST_NAME_ENV)
	raftObj, raftId, transportManager, err := myraft.NewRaft(logger, isBootstrapHost, myHostname, fsm)
	if err != nil {
		logger.Sugar().Errorf("Failed creating new Raft protocol; reason: %v", err)
		retcode = 1
		return
	}
	defer func() {
		logger.Info("Shutting down Raft now")
		err := raftObj.Shutdown().Error()
		if err != nil {
			logger.Sugar().Errorf("Failed shutting down Raft; reason: %v", err)
		} else {
			logger.Info("Successfully shut down Raft")
		}
	}()

	lis, cleanupListener, err := grpctools.NewTcpListener(logger)
	if err != nil {
		logger.Sugar().Errorf("failed creating new listener; reason: %v", err)
		retcode = 1
		return
	}
	defer cleanupListener()

	grpcServer, cleanupGrpcServer := grpctools.NewServer(logger)
	defer cleanupGrpcServer()

	// Register gRPC functions before serving
	logger.Info("Registering Raft functions to the gRPC server")
	transportManager.Register(grpcServer) // For Raft to communicate
	raft_api.RegisterGrpcFuncs(logger, grpcServer, raftObj)

	/*
		Everything up to here can just be killed directly. After this, cleaning
		up becomes very important, in particular leaving the Raft cluster.
	*/
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)

	go func() {
		logger.Sugar().Infof("Beginning to serve gRPC server listening at '%s'", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			logger.Sugar().Errorf("Failed to serve gRPC; reason: %v", err)
			quit <- syscall.SIGINT
		}
	}()

	// Letting all nodes join the cluster
	if !isBootstrapHost {
		// Ask Raft leader whether we can join the cluster
		succeededJoiningCluster := false
		// Could be that some hosts are just leaving the cluster, so we'll try a few
		for _, startingHost := range currentHosts {
			if startingHost == myHostname {
				continue
			}
			err = raft_api.RequestJoinCluster(
				logger,
				startingHost,
				raftId,
				conf.GetServerGRPCAddress(myHostname),
			)
			if err != nil {
				logger.Sugar().Errorf("Failed to request to join the Raft cluster using starting host '%s'; reason: %v",
					startingHost,
					err,
				)
				continue
			}
			succeededJoiningCluster = true
			break
		}
		if !succeededJoiningCluster {
			logger.Sugar().Errorf("Failed to request to join the Raft cluster; reason: %v", err)
			retcode = 1
			return
		}
	}

	logger.Info("The Raft cluster is set up and the followers should have joined by now.")

	/*
		Inform the cluster that we're shutting down
		Leaving the Raft cluster also stops the gRPC server from receiving heartbeats.
		This makes it easier for it to shut down gracefully.
		It also puts the Raft leader at less risk to lose the election.
	*/
	defer raft_api.RequestLeaveCluster(logger, raftObj, raftId, true)

	var interruptWorkerChannel chan bool
	leaderGoRoutine := sync.Mutex{} // Leader can directly win itself again
	logger.Info("Starting to subscribe to leader elections in order to start leader actions if elected")
	for {
		leaderAddress, leaderId := raftObj.LeaderWithID()
		logger.Sugar().Infof("Current leader id: %v; Leader address: %v", leaderId, leaderAddress)

		select {
		case receivedSignal := <-quit:
			logger.Sugar().Warnf("Received signal '%s'", receivedSignal)
			if interruptWorkerChannel != nil {
				interruptWorkerChannel <- true
			}
			return

		// This will just activate if *this node* wins/loses an election
		case isLeaderChannel := <-raftObj.LeaderCh():
			if !isLeaderChannel {
				logger.Warn("We have lost the Raft leadership")
				if interruptWorkerChannel != nil {
					interruptWorkerChannel <- true
				}
				continue
			}

			logger.Info("We have won the Raft leadership. Sleeping a bit to make sure any ex-leader has stopped acting.")
			time.Sleep(5 * time.Second)

			leaderGoRoutine.Lock()

			interruptWorkerChannel = make(chan bool, 1)
			go func() {
				if err := reconciliation.RunLoop(
					logger,
					raftObj,
					fsm,
					interruptWorkerChannel,
					httpClient,
				); err != nil {
					logger.Sugar().Errorf("Failed running the reconciliation loop; reason: %v", err)

					// We may have run into an error because we lost leadership
					_, leaderId := raftObj.LeaderWithID()
					if raftId == leaderId {
						quit <- syscall.SIGINT
					}

				} else {
					logger.Warn("The reconciliation worker has stopped")
				}
				leaderGoRoutine.Unlock()
			}()

		case <-time.After(4 * time.Second):
			continue

		}
	}
}

func setupLogger() (*zap.Logger, error) {
	atom := zap.NewAtomicLevel()
	atom.SetLevel(zap.DebugLevel)

	return zap.Config{
		Level:            atom,
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			CallerKey:    "caller",
			TimeKey:      "time",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeTime:   zapcore.RFC3339NanoTimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}.Build()
}
