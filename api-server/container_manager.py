import copy
import os
import os.path
import docker
import uuid


IMAGE_TO_SPAWN = os.environ.get("IMAGE_TO_SPAWN")
API_SERVER_BASE_URL = os.environ.get("API_SERVER_BASE_URL")

client = docker.from_env()

environment = {"BOOTSTRAP_HOST": "false", "API_SERVER_BASE_URL": API_SERVER_BASE_URL}

networks = client.networks.list()
all_network_attrs = [network.attrs for network in networks]
attacheable_networks = list(
    filter(lambda x: (x["Attachable"] == True), all_network_attrs)
)
compose_network = attacheable_networks[0]["Name"]
print(f"Name of compose_network: '{compose_network}'")


def get_container_names_in_network():
    # Only returns running containers
    containers = client.containers.list()
    containers_in_network = filter_containers_in_network(containers)
    container_names = extract_container_names(containers_in_network)
    unrelated = ["api-server"]
    return [container for container in container_names if container not in unrelated]


def filter_containers_in_network(containers):
    """
    Filter containers which are using the Docker compose network
    """
    return [
        container
        for container in containers
        if "scalable-raft-demo_default"
        in container.attrs["NetworkSettings"]["Networks"]
    ]


def extract_container_names(containers):
    """
    Read container names from Container class
    """
    return [container.attrs["Name"].removeprefix("/") for container in containers]


def stop_container(container_name, delete=False):
    container = client.containers.get(container_name)
    print(f"Stopping container '{container_name}'")
    container.stop()
    if delete:
        print(f"Removing container '{container_name}'")
        container.remove()


def create_containers(num_containers):
    for i in range(num_containers):
        desired_container_name = f"raft-node-{i}-{str(uuid.uuid4())[0:5]}"
        new_env = copy.deepcopy(environment)
        new_env["HOST_NAME"] = desired_container_name
        client.containers.run(
            IMAGE_TO_SPAWN,
            name=desired_container_name,
            detach=True,
            environment=new_env,
            network=compose_network,
            # restart_policy={"Name": "on-failure", "MaximumRetryCount": 20}
        )


def remove_containers(container_names):
    for container in container_names:
        stop_container(container, True)


def post_desired_container(num_desired_container=1):
    print(f"Got request for {num_desired_container} desired containers")

    container_names_in_network = get_container_names_in_network()
    num_running_now = len(container_names_in_network)
    print(f"Number of containers running now: {num_running_now}")
    containers_to_create = num_desired_container - num_running_now

    if containers_to_create > 0:
        print(f"Containers to create: {containers_to_create}")
        create_containers(containers_to_create)
    elif containers_to_create < 0:
        containers_to_remove = -containers_to_create
        print(f"Containers to remove: {containers_to_remove}")
        remove_containers(container_names_in_network[0:containers_to_remove])
    return
