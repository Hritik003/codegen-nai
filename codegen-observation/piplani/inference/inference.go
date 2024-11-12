package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	"github.com/nutanix-core/nai-api/common/response"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	openai "github.com/sashabaranov/go-openai"
)

// InferenceController struct for inference requests
type InferenceController struct {
	v1Route            *gin.RouterGroup
	logger             logger.Logger
	metrics            service.IMetricInstrumentationService
	inferenceService   service.IInferenceService
	inferenceValidator middleware.IInferenceValidator
}

// NewInferenceController creates and initiates the route
func NewInferenceController(v1Route *gin.RouterGroup, logger logger.Logger, metrics service.IMetricInstrumentationService, inferenceService service.IInferenceService, inferenceValidator middleware.IInferenceValidator) *InferenceController {
	controller := &InferenceController{v1Route: v1Route, logger: logger, metrics: metrics, inferenceService: inferenceService, inferenceValidator: inferenceValidator}
	controller.route()
	return controller
}

// Route inference requests to the correct function
func (ic *InferenceController) route() {
	route := ic.v1Route.Group("/", ic.inferenceValidator.ValidateInference())
	route.POST("completions", ic.Completion)
	route.POST("chat/completions", ic.ChatCompletion)
}

// Completion godoc
//
//	@Summary		completion
//	@Description	create a new completions request
//	@Tags			inference
//	@Accept			json
//	@Produce		json
//	@Param			request			body		openai.CompletionRequest			true	"new completions object"
//	@Param			Authorization	header		string								true	"apikey sent via headers"	default(Bearer <Add access token here>)
//	@Success		200				{object}	openai.CompletionResponse			"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel	"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Failure		504				{object}	response.HTTPFailureResponseModel	"gateway timeout error response"
//	@Router			/v1/completions [post]
//
// bypassing duplicate code (dupl) linting
//
//nolint:dupl
func (ic *InferenceController) Completion(c *gin.Context) {
	errMsg := "Completions request failed"
	succMsg := "Completions request success"
	before := c.GetTime(constants.PreValidationTimestamp)
	completionsBody, exists := c.Get("completionsRequest")
	if !exists {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Err: &e.Error{Type: e.NotFoundError, Msg: errMsg + ": failed retrieving request body", Log: "failed retrieving request body"}})
		return
	}

	completionRequest, ok := completionsBody.(openai.CompletionRequest)
	if !ok {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Err: &e.Error{Type: e.BindingError, Msg: errMsg, Log: "failed parsing request body"}})
		return
	}

	engine, err := ic.getEngineParam(c, errMsg)
	if err != nil {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
		return
	}

	if completionRequest.Stream {
		responseID := uuid.New().String()
		stream, err := ic.inferenceService.CompletionStream(completionRequest, engine)
		if err != nil {
			ic.metrics.RecordInferenceMetrics(c, completionRequest.Model, "failure", time.Since(before).Milliseconds())
			response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
			return
		}

		defer stream.Close() //nolint:errcheck

		c.Stream(func(w io.Writer) bool {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				_, _ = w.Write([]byte("data: [DONE]\n\n"))

				ic.logger.Debug(fmt.Sprintf("Stream finished for request on endpoint: %s", completionRequest.Model))
				ic.metrics.RecordInferenceMetrics(c, completionRequest.Model, "success", time.Since(before).Milliseconds())
				return false // Here setting false to exit the streaming process
			}

			if err != nil {
				ic.logger.Debug(fmt.Sprintf("Stream error for request on endpoint: %s; error %v", completionRequest.Model, err))
				ic.metrics.RecordInferenceMetrics(c, completionRequest.Model, "failure", time.Since(before).Milliseconds())
				return false // Here setting false to exit the streaming process
			}

			if response.ID == "" {
				response.ID = responseID
			}
			response.Model = completionRequest.Model
			ic.streamResponse(w, response)
			return true // Here setting true to not exit and continue streaming process
		})

		return
	}

	completionResponse, err := ic.inferenceService.Completion(completionRequest, engine)
	if err != nil {
		ic.metrics.RecordInferenceMetrics(c, completionRequest.Model, "failure", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
		return
	}

	if completionResponse.ID == "" {
		completionResponse.ID = uuid.New().String()
	}
	completionResponse.Model = completionRequest.Model
	ic.metrics.RecordInferenceMetrics(c, completionRequest.Model, "success", time.Since(before).Milliseconds())

	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Data: completionResponse, Raw: true})
}

// ChatCompletion godoc
//
//	@Summary		chatCompletion
//	@Description	create a new chat completion request
//	@Tags			inference
//	@Accept			json
//	@Produce		json
//	@Param			request			body		openai.ChatCompletionRequest		true	"new chat completions object"
//	@Param			Authorization	header		string								true	"apikey sent via headers"	default(Bearer <Add access token here>)
//	@Success		200				{object}	openai.ChatCompletionResponse		"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel	"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Failure		504				{object}	response.HTTPFailureResponseModel	"gateway timeout error response"
//	@Router			/v1/chat/completions [post]
//
// bypassing duplicate code (dupl) linting
//
//nolint:dupl
func (ic *InferenceController) ChatCompletion(c *gin.Context) {
	errMsg := "Chat completions request failed"
	succMsg := "Chat completions request success"
	before := c.GetTime(constants.PreValidationTimestamp)
	chatBody, exists := c.Get("chatRequest")
	if !exists {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Err: &e.Error{Type: e.NotFoundError, Msg: errMsg + ": failed retrieving request body", Log: "failed retrieving request body"}})
		return
	}

	chatRequest, ok := chatBody.(openai.ChatCompletionRequest)
	if !ok {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Err: &e.Error{Type: e.BindingError, Msg: errMsg, Log: "failed parsing request body"}})
		return
	}

	engine, err := ic.getEngineParam(c, errMsg)
	if err != nil {
		ic.metrics.RecordInferenceMetrics(c, "", "invalid", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
		return
	}

	if chatRequest.Stream {
		responseID := uuid.New().String()
		stream, err := ic.inferenceService.ChatCompletionStream(chatRequest, engine)
		if err != nil {
			ic.metrics.RecordInferenceMetrics(c, chatRequest.Model, "failure", time.Since(before).Milliseconds())
			response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
			return
		}

		defer stream.Close() //nolint:errcheck

		c.Stream(func(w io.Writer) bool {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				_, _ = w.Write([]byte("data: [DONE]\n\n"))

				ic.logger.Debug(fmt.Sprintf("Stream finished for request on endpoint: %s", chatRequest.Model))
				ic.metrics.RecordInferenceMetrics(c, chatRequest.Model, "success", time.Since(before).Milliseconds())
				return false // Here setting false to exit the streaming process
			}

			if err != nil {
				ic.logger.Debug(fmt.Sprintf("Stream error for request on endpoint: %s; error %v", chatRequest.Model, err))
				ic.metrics.RecordInferenceMetrics(c, chatRequest.Model, "failure", time.Since(before).Milliseconds())
				return false // Here setting false to exit the streaming process
			}

			if response.ID == "" {
				response.ID = responseID
			}
			response.Model = chatRequest.Model
			ic.streamResponse(w, response)
			return true // Here setting true to not exit and continue streaming process
		})

		return
	}

	chatResponse, err := ic.inferenceService.ChatCompletion(chatRequest, engine)
	if err != nil {
		ic.metrics.RecordInferenceMetrics(c, chatRequest.Model, "failure", time.Since(before).Milliseconds())
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, Err: err})
		return
	}

	if chatResponse.ID == "" {
		chatResponse.ID = uuid.New().String()
	}

	chatResponse.Model = chatRequest.Model
	ic.metrics.RecordInferenceMetrics(c, chatRequest.Model, "success", time.Since(before).Milliseconds())

	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ic.logger, SuccMsg: succMsg, Data: chatResponse, Raw: true})
}

func (ic *InferenceController) streamResponse(w io.Writer, response any) {
	// As response is received from OpenAI golang SDK, we can not manipulate the error on JSON Encode
	_, _ = w.Write([]byte("data: "))
	_ = json.NewEncoder(w).Encode(response) //nolint:errchkjson
	_, _ = w.Write([]byte("\n"))
}

func (ic *InferenceController) getEngineParam(c *gin.Context, errMsg string) (enum.Engine, *e.Error) {
	// This is set internally during inference validator
	engineParam, exists := c.Get(constants.EndpointEngine)
	if !exists {
		msg := "failed retrieving engine param"
		err := &e.Error{Type: e.NotFoundError, Msg: errMsg + ": " + msg, Log: msg}
		return "", err
	}

	engine, ok := engineParam.(enum.Engine)
	if !ok {
		err := &e.Error{Type: e.ParsingError, Msg: errMsg, Log: "engine should be one of tgi/vllm/nim. Please specify a valid engine"}
		return "", err
	}

	return engine, nil
}
