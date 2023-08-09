# Demo of a Scalable, Self-Healing Raft Cluster

This is a demo that shows how Raft can be used to automatise adding and removing nodes to the cluster. The code here attempts to minimise the risk of a cluster failure - this is when a majority of nodes in the cluster are dead/unreachable.

## Architecture

<p align="center">
  <img src="https://github.com/olapiv/scalable-raft-demo/assets/31848129/4b02e364-758f-4c55-826d-e170235d7708" alt="Demo architecture" height="500">
</p>

* The leader asks the api-server for a desired state
* The leader asks the api-server to add/remove containers to reach the desired state
* New followers ask the api-server who is in the cluster
* New followers ask around who the leader is
* New followers ask the leader whether they can join the cluster
* Dying followers ask the leader whether it can remove them from the cluster
* The leader continuously checks the cluster state to check whether it has diverged from the desired state

## Quickstart

1. Start the cluster: `docker-compose up`
2. Kill any of the Raft containers (not all)
3. Watch the cluster self-heal
4. Change `NumberReplicas` in [desired_state.json](desired_state.json)
5. Watch the cluster scale

***NOTE:*** The api-server has the Docker socket mounted to it. This means it can create containers by itself.
