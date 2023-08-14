# Demo of a Scalable, Self-Healing Raft Cluster

This is a demo for the Medium blog post [How to Design a Self-Healing, Dynamic-Size Raft Cluster in Go](https://medium.com/@vincent.lohse/how-to-design-a-self-healing-dynamic-size-raft-cluster-in-go-1c2504af8099). It shows how to use the API of [Hashicorp's Raft library in Go](https://github.com/hashicorp/raft) to automatise adding and removing Raft peers to the cluster. The code here attempts to minimise the risk of a cluster failure - this is when a majority of nodes in the cluster are dead/unreachable.

## Quickstart

1. Start the cluster: `docker-compose up`
2. Kill any of the Raft containers (not all)
3. Watch the cluster self-heal
4. Change `NumberReplicas` in [desired_state.json](desired_state.json)
5. Watch the cluster scale

***NOTE:*** The api-server has the Docker socket mounted to it. This means it can create containers by itself.

## Architecture

<p align="center">
  <img src="https://github.com/olapiv/scalable-raft-demo/assets/31848129/27490267-11ed-426d-a2fa-011d68ec5b97" alt="Demo architecture" height="400">
</p>

* The leader asks the api-server for a desired state
* The leader continuously checks the cluster state to check whether it has diverged from the desired state
* The leader asks the api-server to add/remove containers to reach the desired state
* New followers ask the api-server who is in the cluster
* New followers ask around who the leader is
* New followers ask the leader whether they can join the cluster
* Dying followers ask the leader whether it can remove them from the cluster
