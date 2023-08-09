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
			desiredState, err := httpClient.GetDesiredState()
			if err != nil {
				return err
			}

			currentState, err := myFsm.GetState()
			if err != nil {
				return err
			}

			if currentState.Id == desiredState.Id {
				continue
			}

			log.Sugar().Infof("We received a new desired state with id '%d'", desiredState.Id)

			// Persisting state now
			err = myraft.MarshallAndApply(raftObj, desiredState)
			if err != nil {
				return err
			}

			future := raftObj.GetConfiguration()
			if err := future.Error(); err != nil {
				return err
			}
			config := future.Configuration()

			if desiredState.NumberReplicas == uint8(len(config.Servers)) {
				continue
			}

			err = httpClient.PostDesiredHosts(desiredState.NumberReplicas)
			if err != nil {
				return err
			}
			time.Sleep(3 * time.Second)
		}
	}
}
