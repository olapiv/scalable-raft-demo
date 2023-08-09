package http_client

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/olapiv/scalable-raft-demo/conf"
	"go.uber.org/zap"
)

type HttpClient struct {
	httpClient               *http.Client
	log                      *zap.Logger
	getDesiredStateEndpoint  string
	postDesiredHostsEndpoint string
	getCurrentHostsEndpoint  string
}

const RELATIVE_ENDPOINT_GET_DESIRED_STATE = "desired_state"
const RELATIVE_ENDPOINT_POST_DESIRED_HOSTS = "desired_hosts"
const RELATIVE_ENDPOINT_GET_CURRENT_HOSTS = "current_hosts"

func New(log *zap.Logger) (*HttpClient, error) {
	getDesiredStateEndpoint := conf.GenerateFullApiServerUrl(RELATIVE_ENDPOINT_GET_DESIRED_STATE)
	postDesiredHostsEndpoint := conf.GenerateFullApiServerUrl(RELATIVE_ENDPOINT_POST_DESIRED_HOSTS)
	getCurrentHostsEndpoint := conf.GenerateFullApiServerUrl(RELATIVE_ENDPOINT_GET_CURRENT_HOSTS)

	self := &HttpClient{
		httpClient:               &http.Client{Timeout: 60 * time.Second},
		log:                      log,
		getDesiredStateEndpoint:  getDesiredStateEndpoint,
		postDesiredHostsEndpoint: postDesiredHostsEndpoint,
		getCurrentHostsEndpoint:  getCurrentHostsEndpoint,
	}

	return self, nil
}

func (client *HttpClient) createPostRequest(url string, data io.Reader) (*http.Request, error) {
	return client.createRequest(url, http.MethodPost, data)
}

func (client *HttpClient) createGetRequest(url string) (*http.Request, error) {
	return client.createRequest(url, http.MethodGet, nil)
}

func (client *HttpClient) createRequest(url string, httpMethod string, data io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(httpMethod, url, data)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")
	return request, nil
}

func (client *HttpClient) doHttpRequest(request *http.Request) ([]byte, error) {
	var err error
	var response *http.Response
	try := 0
	for {
		/*
			We only use the HTTP client towards the api-server. Since we always expect it to be
			running, we only implement a very simple, non-configurable retry mechanism here.
		*/
		response, err = client.httpClient.Do(request)
		if err != nil {
			client.log.Sugar().Errorf("Failed running HTTP request on attempt %d; reason: %v", try, err)
			if try < 5 {
				try++
				client.log.Info("Waiting 2 seconds before retrying a request")
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, err
		}
		break
	}
	defer response.Body.Close()
	content, errReadContent := io.ReadAll(response.Body)
	if response.StatusCode != 200 {
		mainMsg := fmt.Sprintf("http status code for url %s with httpMethod %s is not 200 (%d)", request.URL.String(), request.Method, response.StatusCode)
		var extraMsg string
		if errReadContent == nil {
			extraMsg = fmt.Sprintf("content of response body: '%s'", string(content))
		} else {
			extraMsg = fmt.Sprintf("error reading response body; reason: %s", errReadContent.Error())
		}
		errMsg := fmt.Errorf("%s; %s", mainMsg, extraMsg)
		return nil, errMsg
	}
	return content, nil
}
