package http_client

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/olapiv/scalable-raft-demo/myraft"
)

func (client *HttpClient) GetDesiredState() (newDesiredState *myraft.DesiredState, err error) {
	client.log.Info("Requesting a new desired state")
	getDesiredStateRequest, err := client.createGetRequest(client.getDesiredStateEndpoint)
	if err != nil {
		err = fmt.Errorf("cannot create request to get desired state; reason: %v", err)
		return
	}
	body, err := client.doHttpRequest(getDesiredStateRequest)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &newDesiredState)
	return
}

func (client *HttpClient) PostDesiredHosts(numDesiredHosts uint8) (err error) {
	data := map[string]interface{}{
		"numDesiredHosts": numDesiredHosts,
	}
	marshalled, err := json.Marshal(data)
	if err != nil {
		return err
	}
	payload := bytes.NewReader(marshalled)

	// Just fail if the reconciliation state is still working
	proposeDesiredStateRequest, err := client.createPostRequest(client.postDesiredHostsEndpoint, payload)
	if err != nil {
		err = fmt.Errorf("cannot create request to decline desired state; reason: %v", err)
		return
	}
	_, err = client.doHttpRequest(proposeDesiredStateRequest)
	return
}

func (client *HttpClient) GetCurrentHosts() ([]string, error) {
	getCurrentVmsRequest, err := client.createGetRequest(client.getCurrentHostsEndpoint)
	if err != nil {
		err = fmt.Errorf("cannot create request to get current hosts; reason: %w", err)
		return nil, err
	}
	body, err := client.doHttpRequest(getCurrentVmsRequest)
	if err != nil {
		return nil, err
	}
	response := []string{}
	err = json.Unmarshal(body, &response)

	return response, err
}
