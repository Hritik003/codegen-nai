package client

import (
	"context"
	"net/http"
	"time"

	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
)

// IHealthClient interface contains methods to fetch inference endpoint health
type IHealthClient interface {
	CheckHealth(healthCheckURL string) enum.ServiceHealthStatusCode
}

// healthClient struct client executes health api calls on inference endpoints
type healthClient struct {
	client IClient
}

// NewHealthClient instantiates Inference client
func NewHealthClient(client IClient) IHealthClient {
	return &healthClient{
		client: client,
	}
}

// CheckHealth executes health api call for a given inference endpoint
func (ic *healthClient) CheckHealth(healthCheckURL string) enum.ServiceHealthStatusCode {
	ic.client.SetTimeout(constants.ClientTimeout * time.Second)

	var status enum.ServiceHealthStatusCode

	// if service is unreachable/unhealthy, retry for MaxServiceHealthRetries times
	for retry := 1; retry <= constants.MaxServiceHealthAttempts; retry++ {
		status = ic.checkHealthInternal(healthCheckURL)

		if status == enum.HealthyStatusCode {
			return status
		}
		// retry if error occurred while checking health or service is unhealthy
		// sleep before retrying, do not sleep after last retry
		if retry < constants.MaxServiceHealthAttempts {
			time.Sleep(constants.ServiceHealthCheckRetryDelay * time.Second)
		}
	}
	return status
}

func (ic *healthClient) checkHealthInternal(healthCheckURL string) enum.ServiceHealthStatusCode {
	req, err := http.NewRequest(http.MethodGet, healthCheckURL, nil)
	if err != nil {
		// returning status as unknown as http request creation failed, hence the status is unknown
		return enum.UnknownStatusCode
	}

	resp, _, err := ic.client.Do(context.Background(), req)

	// endpoint health is critical as health api failed
	if err != nil {
		return enum.CriticalStatusCode
	}

	// no error while executing request
	defer resp.Body.Close() //nolint:errcheck

	// check response status code
	if resp.StatusCode != http.StatusOK {
		return enum.CriticalStatusCode
	}

	return enum.HealthyStatusCode
}
