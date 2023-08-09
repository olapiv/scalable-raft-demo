# Raft API

This package makes a subset of methods that Raft exposes available via gRPC calls. This is important since e.g. only the leader can add nodes to a Raft cluster. Thereby, the followers need some kind of API towards the leader, where they can request to join the cluster. The same can be used for a follower that is planning to drop out of the cluster - it can let the leader know beforehand. This means that the leader will not potentially believe that it has lost a majority and run for a re-election. The API can also be used to simply query *any* node in the cluster who the leader actually is.

This package was inspired by https://github.com/Jille/raftadmin.
