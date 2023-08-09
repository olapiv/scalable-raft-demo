import json5
from threading import Lock


desired_state_file = "./desired_state.json"

public_state_lock = Lock()
desired_state_lock = Lock()

def read_desired_state():
    desired_state_lock.acquire()
    with open(desired_state_file) as json_file:
        new_desired_state = json5.load(json_file)
    desired_state_lock.release()
    return new_desired_state
