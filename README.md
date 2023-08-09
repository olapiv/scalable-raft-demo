# Demo of a Scalable, Self-Healing Raft Cluster

This is a demo that shows how Raft can be used to automatise adding and removing nodes to the cluster. The code here attempts to minimise the risk of a cluster failure, where a majority of nodes in the cluster are unreachable.

The architecture is as follows:

1. The leader asks the api-server for a desired state
2. The leader asks the api-server to add/remove containers to reach the desired state
3. New followers ask the api-server who is in the cluster
4. New followers ask around who the leader is
5. New followers ask the leader whether they can join the cluster
6. Dying followers ask the leader whether it can remove them from the cluster
7. The leader continuously checks the cluster state to check whether it has diverged from the desired state

## Quickstart

1. Start the cluster: `docker-compose up`
2. Kill any of the Raft containers (not all)
3. Watch the cluster self-heal
4. Change `NumberReplicas` in [desired_state.json](desired_state.json)
5. Watch the cluster scale

***NOTE:*** The api-server has the Docker socket mounted to it. This means it can create containers by itself.
