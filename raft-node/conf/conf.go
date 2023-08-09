package conf

import (
	"fmt"
	"os"
)

const GRPC_LISTEN_PORT = 8000
const BOOTSTRAP_HOST_ENV = "BOOTSTRAP_HOST"
const API_SERVER_BASE_ENDPOINT_ENV = "API_SERVER_BASE_URL"
const HOST_NAME_ENV = "HOST_NAME" // Set via container names

// The addressHost parameter can be left empty
func GetServerGRPCAddress(addressHost string) string {
	return fmt.Sprintf("%s:%d", addressHost, GRPC_LISTEN_PORT)
}

func GenerateFullApiServerUrl(relativeUrl string) string {
	baseEndpoint := os.Getenv(API_SERVER_BASE_ENDPOINT_ENV)
	return fmt.Sprintf("%s/%s", baseEndpoint, relativeUrl)
}
