package v1_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	v1 "github.com/nutanix-core/nai-api/iep/api/v1"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	mock_middleware "github.com/nutanix-core/nai-api/iep/mocks/middleware"
	mock_service "github.com/nutanix-core/nai-api/iep/mocks/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"
)

func getCompletionRequest(stream bool) openai.CompletionRequest {
	return openai.CompletionRequest{
		Model:  "endpoint_id",
		Prompt: "text",
		Stream: stream,
	}
}

func getChatRequest(stream bool) openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model: "endpoint_id",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: "role",
				MultiContent: []openai.ChatMessagePart{
					{
						Type: "message_type",
						Text: "text",
					},
				},
			},
		},
		Stream: stream,
	}
}

var _ = Describe("Test Inference Controller", func() {

	var (
		mockCtrl                   *gomock.Controller
		mockInferenceService       *mock_service.MockIInferenceService
		mockInferenceValidator     *mock_middleware.MockIInferenceValidator
		mockKserveService          *mock_service.MockIKServeService
		mockInstrumentationService *mock_service.MockIMetricInstrumentationService
		before                     time.Time
		logger                     = logger.NewZAPLogger()
		mockAPIKey                 = "mockValidAPIKey" // #nosec G101
		name                       = "endpoint_id"
		engine                     = enum.VLLMEngine

		correctCompletionRequest = `
			{
				"model": "endpoint_id",
				"prompt": "text"
			}`
		completionStreamRequest = `
			{
				"model": "endpoint_id",
				"prompt": "text",
				"stream": "true"
			}`
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockCtrl = gomock.NewController(GinkgoT())
		mockInferenceService = mock_service.NewMockIInferenceService(mockCtrl)
		mockInferenceValidator = mock_middleware.NewMockIInferenceValidator(mockCtrl)
		mockKserveService = mock_service.NewMockIKServeService(mockCtrl)
		mockInstrumentationService = mock_service.NewMockIMetricInstrumentationService(mockCtrl)
		validHandler := func(c *gin.Context) {
			c.Next()
		}
		mockInferenceValidator.EXPECT().ValidateInference().Return(validHandler).Times(1)

	})

	Context("Test OpenAI Completion Request", func() {
		It("OpenAI Completion Successful", func() {
			completionRequest := getCompletionRequest(false)
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "success", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().Completion(completionRequest, engine).Return(openai.CompletionResponse{}, nil).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Completion Stream Throw error from service", func() {
			completionRequest := getCompletionRequest(true)
			validContext, router := getContext("api/v1/completions", completionStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "failure", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().CompletionStream(completionRequest, engine).Return(nil, &e.Error{Type: e.GenericError, InternalErr: errors.New("Completion inference failed")}).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("OpenAI Completion Stream non EOF Error", func() {
			completionRequest := getCompletionRequest(true)
			validContext, router := getContext("api/v1/completions", completionStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "failure", time.Since(before).Milliseconds())
			body := openai.CompletionResponse{
				ID:    "requestId",
				Model: "endpoint_id",
			}

			mockResponses := []openai.CompletionResponse{body, body, body}

			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/openai/v1/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")

				for _, resp := range mockResponses {
					fmt.Fprintf(w, "data: %v\n\n", resp)
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.VLLMEngine).Return(fmt.Sprintf("%s/openai/v1", host))
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()

		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Completion Stream Successful", func() {
			completionRequest := getCompletionRequest(true)
			validContext, router := getContext("api/v1/completions", completionStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "success", time.Since(before).Milliseconds())
			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/openai/v1/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")

				// Send test responses
				dataBytes := []byte{}

				data := `{"id":"1","object":"completion","created":1598069254,"model":"endpoint_id","choices":[{"text":"response1","finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				data = `{"id":"2","object":"completion","created":1598069255,"model":"endpoint_id","choices":[{"text":"response2","finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				dataBytes = append(dataBytes, []byte("data: [DONE]\n\n")...)

				_, err := w.Write(dataBytes)
				if err != nil {
					logger.Error(err.Error())
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.VLLMEngine).Return(fmt.Sprintf("%s/openai/v1", host))
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()

		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Completion Stream Successful, response with ID empty", func() {
			completionRequest := getCompletionRequest(true)
			validContext, router := getContext("api/v1/completions", completionStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "success", time.Since(before).Milliseconds())
			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/openai/v1/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")

				// Send test responses
				dataBytes := []byte{}

				data := `{"id":"","object":"completion","created":1598069254,"model":"endpoint_id","choices":[{"text":"response1","finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				data = `{"id":"","object":"completion","created":1598069255,"model":"endpoint_id","choices":[{"text":"response2","finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				dataBytes = append(dataBytes, []byte("data: [DONE]\n\n")...)

				_, err := w.Write(dataBytes)
				if err != nil {
					logger.Error(err.Error())
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.VLLMEngine).Return(fmt.Sprintf("%s/openai/v1", host))
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()

		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Completion Successful, For Web UI", func() {
			completionRequest := getCompletionRequest(false)
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Request.Header.Set(constants.ClientType, "ui")
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "success", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().Completion(completionRequest, engine).Return(openai.CompletionResponse{}, nil).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("OpenAI Completion Unsuccessful: completionsRequest not set in context", func() {
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusNotFound))
		})

		It("OpenAI Completion Unsuccessful: completionsRequest not of type openai.CompletionRequest", func() {
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("completionsRequest", getCreateCatalog()) // wrong type
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("OpenAI Completion Unsuccessful: endpointEngine not set in context", func() {
			completionRequest := getCompletionRequest(false)
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusNotFound))
		})

		It("OpenAI Completion Unsuccessful: endpointEngine not of type enum.Engine", func() {
			completionRequest := getCompletionRequest(false)
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set("endpointEngine", "wrong") // wrong type
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("OpenAI Completion Unsuccessful: InferenceService error", func() {
			completionRequest := getCompletionRequest(false)
			validContext, router := getContext("api/v1/completions", correctCompletionRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("completionsRequest", completionRequest)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, completionRequest.Model, "failure", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().Completion(completionRequest, engine).Return(openai.CompletionResponse{}, &e.Error{Type: e.GenericError, InternalErr: errors.New("Completion inference failed")}).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.Completion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	Context("Test OpenAI Chat Completion Request", func() {
		chatRequest := `
			{
				"model": "endpoint_id",
				"messages": [
						{
								"role": "role",
								"content": "content",
								"multi_content": [
										{
												"type": "message_type",
												"text": "text"
										}
								]
						}
				]
			}`

		chatStreamRequest := `
			{
				"model": "endpoint_id",
				"messages": [
						{
								"role": "role",
								"content": "content",
								"multi_content": [
										{
												"type": "message_type",
												"text": "text"
										}
								]
						}
				],
				stream: true
			}`

		It("OpenAI Chat Completion Successful", func() {
			chatCompletionRequest := getChatRequest(false)
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "success", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().ChatCompletion(chatCompletionRequest, engine).Return(openai.ChatCompletionResponse{}, nil).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Chat Completion Stream Throw error from service", func() {
			chatCompletionRequest := getChatRequest(true)
			validContext, router := getContext("api/v1/chat/completions", chatStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "failure", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().ChatCompletionStream(chatCompletionRequest, engine).Return(nil, &e.Error{Type: e.GenericError, InternalErr: errors.New("Chat Completion inference failed")}).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("OpenAI Chat Completion Stream non EOF Error", func() {
			chatCompletionRequest := getChatRequest(true)
			validContext, router := getContext("api/v1/chat/completions", chatStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.TGIEngine)

			body := openai.ChatCompletionResponse{
				ID:    "requestId",
				Model: "endpoint_id",
			}

			mockResponses := []openai.ChatCompletionResponse{body, body, body}

			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")

				for _, resp := range mockResponses {
					fmt.Fprintf(w, "data: %v\n\n", resp)
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.TGIEngine).Return(fmt.Sprintf("%s/v1", host))
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "failure", time.Since(before).Milliseconds())
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()

		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Chat Completion Stream Successful", func() {
			chatCompletionRequest := getChatRequest(true)
			validContext, router := getContext("api/v1/chat/completions", chatStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set("endpointEngine", enum.TGIEngine)

			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")

				// Send test responses
				dataBytes := []byte{}
				data := `{"id":"1","object":"completion","created":1598069254,"model":"endpoint_id","choices":[{"index":0,"delta":{"content":"response1"},"finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				data = `{"id":"2","object":"completion","created":1598069255,"model":"endpoint_id","choices":[{"index":0,"delta":{"content":"response2"},"finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				dataBytes = append(dataBytes, []byte("data: [DONE]\n\n")...)

				_, err := w.Write(dataBytes)
				if err != nil {
					logger.Error(err.Error())
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.TGIEngine).Return(fmt.Sprintf("%s/v1", host))
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "success", time.Since(before).Milliseconds())
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()
		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Chat Completion Stream Successful, response with Id empty", func() {
			chatCompletionRequest := getChatRequest(true)
			validContext, router := getContext("api/v1/chat/completions", chatStreamRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set("endpointEngine", enum.TGIEngine)

			server, teardown := setupOpenAITestServer()
			defer teardown()
			server.RegisterHandler("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")

				// Send test responses
				dataBytes := []byte{}
				data := `{"id":"","object":"completion","created":1598069254,"model":"endpoint_id","choices":[{"index":0,"delta":{"content":"response1"},"finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				data = `{"id":"","object":"completion","created":1598069255,"model":"endpoint_id","choices":[{"index":0,"delta":{"content":"response2"},"finish_reason":"max_tokens"}]}`
				dataBytes = append(dataBytes, []byte("data: "+data+"\n\n")...)

				dataBytes = append(dataBytes, []byte("data: [DONE]\n\n")...)

				_, err := w.Write(dataBytes)
				if err != nil {
					logger.Error(err.Error())
				}
			})

			host, _ := url.Parse(server.OpenAITestServer().URL)

			mockKserveService.EXPECT().GetBaseServiceURL(name, enum.TGIEngine).Return(fmt.Sprintf("%s/v1", host))
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "success", time.Since(before).Milliseconds())
			inferenceService := service.NewInferenceService(mockKserveService, logger)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, inferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)

			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))

			server.OpenAITestServer().Close()
		})

		// bypassing duplicate code (dupl) linting
		//
		//nolint:dupl
		It("OpenAI Chat Completion Successful, For Web UI", func() {
			chatCompletionRequest := getChatRequest(false)
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Request.Header.Set(constants.ClientType, "ui")
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "success", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().ChatCompletion(chatCompletionRequest, engine).Return(openai.ChatCompletionResponse{}, nil).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("OpenAI Chat Completion Unsuccessful: chatRequest not set in context", func() {
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set(constants.PreValidationTimestamp, before)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusNotFound))
		})

		It("OpenAI Chat Completion Unsuccessful: chatRequest not of type openai.ChatCompletionRequest", func() {
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", getCreateCatalog()) // wrong type
			validContext.Set(constants.PreValidationTimestamp, before)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("OpenAI Chat Completion Unsuccessful: endpointEngine not set in context", func() {
			chatCompletionRequest := getChatRequest(false)
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusNotFound))
		})

		It("OpenAI Chat Completion Unsuccessful: endpointEngine not of type enum.Engine", func() {
			chatCompletionRequest := getChatRequest(false)
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", "wrong") // wrong type
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, "", "invalid", time.Since(before).Milliseconds())
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("OpenAI Chat Completion Unsuccessful: InferenceService error", func() {
			chatCompletionRequest := getChatRequest(false)
			validContext, router := getContext("api/v1/chat/completions", chatRequest, "POST")
			validContext.Request.Header.Set("Authorization", "Bearer "+mockAPIKey)
			validContext.Set("chatRequest", chatCompletionRequest)
			validContext.Set(constants.PreValidationTimestamp, before)
			validContext.Set("endpointEngine", enum.VLLMEngine)
			mockInstrumentationService.EXPECT().RecordInferenceMetrics(validContext, chatCompletionRequest.Model, "failure", time.Since(before).Milliseconds())
			mockInferenceService.EXPECT().ChatCompletion(chatCompletionRequest, engine).Return(openai.ChatCompletionResponse{}, &e.Error{Type: e.GenericError, InternalErr: errors.New("Chat Completion inference failed")}).Times(1)
			testInferenceController := v1.NewInferenceController(router.Group("/v1"), logger, mockInstrumentationService, mockInferenceService, mockInferenceValidator)
			testInferenceController.ChatCompletion(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})
})

func setupOpenAITestServer() (server *ServerTest, teardown func()) {
	server = NewTestServer()
	ts := server.OpenAITestServer()
	teardown = ts.Close
	return
}
