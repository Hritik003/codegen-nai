package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	openai "github.com/sashabaranov/go-openai"
)

func TestCompletion(t *testing.T) {
	c := &gin.Context{}
	completionsBody := &openai.CompletionRequest{}
	completionResponse := &openai.CompletionResponse{}
	err := errors.New("error")
	engine := enum.EngineTGI

	c.Set("completionsRequest", completionsBody)
	c.Set(constants.EndpointEngine, engine)

	controller := &InferenceController{v1Route: nil, logger: logger.Logger{}, metrics: service.IMetricInstrumentationService{}, inferenceService: service.IInferenceService{}, inferenceValidator: middleware.IInferenceValidator{}}
	controller.Completion(c)

	if !reflect.DeepEqual(completionResponse, completionResponse) {
		t.Errorf("Expected %v, got %v", completionResponse, completionResponse)
	}

	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func TestChatCompletion(t *testing.T) {
	c := &gin.Context{}
	chatBody := &openai.ChatCompletionRequest{}
	chatResponse := &openai.ChatCompletionResponse{}
	err := errors.New("error")
	engine := enum.EngineTGI

	c.Set("chatRequest", chatBody)
	c.Set(constants.EndpointEngine, engine)

	controller := &InferenceController{v1Route: nil, logger: logger.Logger{}, metrics: service.IMetricInstrumentationService{}, inferenceService: service.IInferenceService{}, inferenceValidator: middleware.IInferenceValidator{}}
	controller.ChatCompletion(c)

	if !reflect.DeepEqual(chatResponse, chatResponse) {
		t.Errorf("Expected %v, got %v", chatResponse, chatResponse)
	}

	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}