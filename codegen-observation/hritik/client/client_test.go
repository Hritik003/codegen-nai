package client_test

import (
	"time"

	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/nutanix-core/nai-api/iep/internal/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test client functions", func() {

	var (
		testServer *httptest.Server
		c          client.IClient
	)

	BeforeEach(func() {
		// Create a new test server
		c = client.NewClient()
	})

	AfterEach(func() {
		// Close the test server after each test if it's running
		if testServer != nil {
			testServer.Close()
		}
	})

	Context("Test Set Timeout function", func() {
		It("Test Set Timeout function", func() {
			c := client.NewClient()
			c.SetTimeout(5 * time.Second)
			Expect(c.GetTimeout()).To(Equal(5 * time.Second))
		})
	})

	Context("Test MakeRequestWithRetry function", func() {
		var (
			url        string
			method     = "POST"
			reqBody    = bytes.NewBufferString(`{"key": "value"}`)
			headers    = map[string]string{"Content-Type": "application/json", "Authorization": "Bearer mock_token"}
			maxRetries = 3
			retryDelay = 1 * time.Second
			ctx        = context.Background()
		)

		It("should succeed on the first attempt", func() {
			// Create a test server that always responds with StatusOK
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Use the test server's URL as the request URL
			url = testServer.URL

			err := c.MakeRequestWithRetry(ctx, url, method, reqBody, headers, maxRetries, retryDelay)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should retry on failure and eventually succeed", func() {
			// Create a test server that fails twice and then succeeds
			attempts := 0
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				if attempts < 3 {
					http.Error(w, "server error", http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))

			// Use the test server's URL as the request URL
			url = testServer.URL

			err := c.MakeRequestWithRetry(ctx, url, method, reqBody, headers, maxRetries, retryDelay)
			Expect(err).ToNot(HaveOccurred())
			Expect(attempts).To(Equal(3)) // Ensure it retries twice before succeeding
		})

		It("should fail after maximum retries", func() {
			// Create a test server that always returns an error
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "server error", http.StatusInternalServerError)
			}))

			// Use the test server's URL as the request URL
			url = testServer.URL

			err := c.MakeRequestWithRetry(ctx, url, method, reqBody, headers, maxRetries, retryDelay)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to make request after multiple retries"))
		})

		It("should return an error when request creation fails", func() {
			// Use an invalid URL to simulate request creation failure
			url = ":"

			err := c.MakeRequestWithRetry(ctx, url, method, reqBody, headers, maxRetries, retryDelay)
			Expect(err).To(HaveOccurred())
		})
	})
})
