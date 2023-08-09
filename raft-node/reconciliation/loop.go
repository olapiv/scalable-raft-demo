package reconciliation

import (
	"time"

	"github.com/hashicorp/raft"
	"github.com/olapiv/scalable-raft-demo/http_client"
	"github.com/olapiv/scalable-raft-demo/myraft"
	"go.uber.org/zap"
)

func RunLoop(
	log *zap.Logger,
	raftObj *raft.Raft,
	myFsm *myraft.MyFSM,
	interruptChannel chan bool,
	httpClient *http_client.HttpClient,
) error {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-interruptChannel:
			return nil
		case <-ticker.C:
			wasCancelled, err := workTowardsDesired(raftObj, myFsm, httpClient, interruptChannel)
			if err != nil {
				return err
			}
			if wasCancelled {
				return nil
			}

			if err := getNewDesired(log, raftObj, myFsm, httpClient); err != nil {
				return err
			}
		}
	}
}

func getNewDesired(
	log *zap.Logger,
	raftObj *raft.Raft,
	myFsm *myraft.MyFSM,
	httpClient *http_client.HttpClient,
) error {
	desiredState, err := httpClient.GetDesiredState()
	if err != nil {
		return err
	}

	currentState, err := myFsm.GetState()
	if err != nil {
		return err
	}

	if currentState != nil && (currentState.NumberReplicas == desiredState.NumberReplicas) {
		log.Info("The requested number of replicas have not changed")
		return nil
	}

	log.Sugar().Infof("We received a new desired number of replicas: '%d'", desiredState.NumberReplicas)

	// Persisting state now
	return myraft.MarshallAndApply(raftObj, desiredState)
}

func workTowardsDesired(
	raftObj *raft.Raft,
	myFsm *myraft.MyFSM,
	httpClient *http_client.HttpClient,
	interruptChannel chan bool,
) (wasCancelled bool, err error) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-interruptChannel:
			return true, nil
		case <-ticker.C:
			currentState, err := myFsm.GetState()
			if err != nil {
				return false, err
			}

			// The state is not initialised yet
			if currentState == nil {
				return false, nil
			}

			future := raftObj.GetConfiguration()
			if err := future.Error(); err != nil {
				return false, err
			}
			config := future.Configuration()

			if currentState.NumberReplicas == uint8(len(config.Servers)) {
				return false, nil
			}

			err = httpClient.PostDesiredHosts(currentState.NumberReplicas)
			if err != nil {
				return false, err
			}
		}
	}
}
