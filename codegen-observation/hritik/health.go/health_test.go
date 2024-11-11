package client_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/client"
	mock_client "github.com/nutanix-core/nai-api/iep/mocks/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Test Inference client methods", func() {
	var _ = Context("Test Get Endpoint Health method", func() {

		var (
			mockCtrl    *gomock.Controller
			mockClient  *mock_client.MockIClient
			endpointURL = "http://endpoint-1.nai-admin.svc.cluster.local/v2/health/live"
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock_client.NewMockIClient(mockCtrl)
		})

		It("HTTP request creation failed", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			healthStatus := inferenceClient.CheckHealth(":www.abv")
			Expect(healthStatus).To(Equal(enum.UnknownStatusCode))
		})

		It("Endpoint status healthy", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"live":true}`)),
			}, []byte(`{"live":true}`), nil).Times(1)
			healthStatus := inferenceClient.CheckHealth(endpointURL)
			Expect(healthStatus).To(Equal(enum.HealthyStatusCode))
		})

		It("Endpoint status critical, after retry", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewBufferString(`{"no healthy upstream"}`)),
			}, []byte(`{"no healthy upstream"}`), nil).Times(constants.MaxServiceHealthAttempts)
			healthStatus := inferenceClient.CheckHealth(endpointURL)
			Expect(healthStatus).To(Equal(enum.CriticalStatusCode))
		})

		It("Endpoint status critical at first, succeeded on retry", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
			Expect(err).ToNot(HaveOccurred())

			gomock.InOrder(
				mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString(`{"no healthy upstream"}`)),
				}, []byte(`{"no healthy upstream"}`), nil).Times(1),
				mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"live":true}`)),
				}, []byte(`{"live":true}`), nil).Times(1),
			)

			healthStatus := inferenceClient.CheckHealth(endpointURL)
			Expect(healthStatus).To(Equal(enum.HealthyStatusCode))
		})

		It("Endpoint status request times out", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
			Expect(err).ToNot(HaveOccurred())
			requestError := &url.Error{
				Op:  http.MethodGet,
				URL: endpointURL,
				Err: &mockTimeoutError{},
			}

			mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{}, nil, requestError).Times(constants.MaxServiceHealthAttempts)
			healthStatus := inferenceClient.CheckHealth(endpointURL)
			Expect(healthStatus).To(Equal(enum.CriticalStatusCode))
		})

		It("Error while checking endpoint status", func() {
			inferenceClient := client.NewHealthClient(mockClient)
			mockClient.EXPECT().SetTimeout(5 * time.Second).Return().Times(1)
			req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
			Expect(err).ToNot(HaveOccurred())
			requestError := &url.Error{
				Op:  http.MethodGet,
				URL: endpointURL,
				Err: errors.New("error while reaching the endpoint"),
			}
			mockClient.EXPECT().Do(context.Background(), req).Return(&http.Response{}, nil, requestError).Times(constants.MaxServiceHealthAttempts)
			healthStatus := inferenceClient.CheckHealth(endpointURL)
			Expect(healthStatus).To(Equal(enum.CriticalStatusCode))
		})
	})
})

type IMockTimeoutError interface {
	Timeout() bool
	Error() string
}

type mockTimeoutError struct {
}

func (e *mockTimeoutError) Error() string { return "request timed out" }
func (e *mockTimeoutError) Timeout() bool {
	return true
}
