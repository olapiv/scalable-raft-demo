package myraft

import (
	"encoding/json"

	"github.com/hashicorp/raft"
)

type Snapshot struct {
	State *DesiredState
}

func (s *Snapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s.State)
	if err != nil {
		return err
	}

	_, err = sink.Write(data)
	if err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *Snapshot) Release() {
}
