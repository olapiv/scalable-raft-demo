package myraft

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"sync"

	"github.com/hashicorp/raft"

	"go.uber.org/zap"
)

/*
	The MyFSM implements the Finite State Machine (FSM) interface
*/

type MyFSM struct {
	log   *zap.Logger
	mu    sync.RWMutex
	state *DesiredState
}

func NewMyFSM(log *zap.Logger) *MyFSM {
	return &MyFSM{
		log:   log,
		mu:    sync.RWMutex{},
		state: &DesiredState{},
	}
}

/*
	Required by Raft FSM interface
*/
func (st *MyFSM) Apply(l *raft.Log) interface{} {

	// This produces A LOT of logs
	// st.log.Sugar().Debugf("Received log: %s", l.Data)

	desiredState := DesiredState{}
	if err := json.Unmarshal(l.Data, &desiredState); err != nil {
		panic(err.Error())
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	st.state = &desiredState
	return nil
}

/*
	Required by Raft FSM interface
*/
func (st *MyFSM) Restore(r io.ReadCloser) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	desiredState := DesiredState{}
	err = json.Unmarshal(b, &desiredState)
	if err != nil {
		return err
	}

	st.state = &desiredState
	return nil
}

/*
	Required by Raft FSM interface
*/
func (fsm *MyFSM) Snapshot() (raft.FSMSnapshot, error) {
	/*
		Make sure that any future calls to f.Apply() don't change the snapshot.

		So basically doing a deep-copy here.
	*/

	state, err := fsm.GetState()
	if err != nil {
		return nil, err
	}

	return &Snapshot{state}, nil
}

// Deep-copy of state
func (fsm *MyFSM) GetState() (*DesiredState, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	data, err := json.Marshal(fsm.state)
	if err != nil {
		return nil, err
	}

	desiredState := DesiredState{}
	err = json.Unmarshal(data, &desiredState)
	if err != nil {
		return nil, err
	}

	return &desiredState, nil
}
