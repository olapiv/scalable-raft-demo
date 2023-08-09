import json
import os
import os.path
from threading import Thread

from flask import Flask, request
from werkzeug.serving import make_server

from utils import read_desired_state
from container_manager import post_desired_container, get_container_names_in_network


PORT = int(os.environ.get("FLASK_RUN_PORT", 8000))

app = Flask(__name__)


@app.route("/desired_state", methods=["GET"])
def get_desired_state():
    new_desired_state = read_desired_state()
    print(
        f"""
Sending desired state:
    {new_desired_state}
"""
    )
    return new_desired_state


@app.route("/desired_hosts", methods=["POST"])
def desired_hosts():
    num_desired_hosts = (
        request.json["numDesiredHosts"] if request.json["numDesiredHosts"] else 1
    )
    # Starting thread to avoid a timeout
    thread = Thread(target=post_desired_container, args=(num_desired_hosts,))
    thread.start()
    return (
        "<p>In the process of creating/removing managed VMs to reach desired state</p>"
    )


@app.route("/current_hosts", methods=["GET"])
def current_hosts():
    all_running_container_names = get_container_names_in_network()
    print(f"all_running_container_names: {all_running_container_names}")
    return json.dumps(all_running_container_names), 200


if __name__ == "__main__":
    server = make_server("0.0.0.0", PORT, app)
    server.serve_forever()
