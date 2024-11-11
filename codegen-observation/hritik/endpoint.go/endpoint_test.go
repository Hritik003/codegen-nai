package v1_test

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	v1 "github.com/nutanix-core/nai-api/iep/api/v1"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	dto "github.com/nutanix-core/nai-api/iep/internal/dto"
	auth "github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/model"
	"github.com/nutanix-core/nai-api/iep/internal/validator"
	mock_middleware "github.com/nutanix-core/nai-api/iep/mocks/middleware"
	mock_service "github.com/nutanix-core/nai-api/iep/mocks/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func getCreateEndpointRequest() dto.CreateEndpointRequest {
	cpu := int64(24)
	memoryInGi := int64(256)
	gpu := int64(1)
	instances := int64(1)
	tgiEngine := enum.TGIEngine
	gpuProduct := "NVIDIA-A100-PCIE-40GB"
	return dto.CreateEndpointRequest{
		Name:         "gpt2-dep1",
		ModelID:      "348967bb-386d-41d0-93cb-ca30f7bbd07d",
		MinInstances: &instances,
		MaxInstances: &instances,
		CPU:          &cpu,
		MemoryinGi:   &memoryInGi,
		GPU:          &gpu,
		Engine:       tgiEngine,
		GPUProduct:   gpuProduct,
		Quantization: enum.Float16,
	}
}

func getCreateEndpointRequestForCPU() dto.CreateEndpointRequest {
	cpu := int64(24)
	memoryInGi := int64(256)
	gpu := int64(0)
	instances := int64(1)
	vllmEngine := enum.VLLMEngine
	return dto.CreateEndpointRequest{
		Name:         "gpt2-dep1",
		ModelID:      "348967bb-386d-41d0-93cb-ca30f7bbd07d",
		MinInstances: &instances,
		MaxInstances: &instances,
		CPU:          &cpu,
		MemoryinGi:   &memoryInGi,
		GPU:          &gpu,
		Engine:       vllmEngine,
		Quantization: enum.Float16,
	}
}

var _ = Describe("Test Endpoint Controller", func() {

	var (
		mockCtrl            *gomock.Controller
		mockEndpointService *mock_service.MockIEndpointService
		mockAuthService     *mock_middleware.MockIAuthenticationMiddleware
		logger              = logger.NewZAPLogger()
		endpointValidator   = validator.NewValidator(logger)
		userIDKey           = "userID"
		userID              = uuid.NewString()
		roleKey             = "role"
		role                = "MLAdmin"
		createdAt           = time.Now()
		userContext         = dto.UserContext{
			UserID: userID,
			Role:   model.MLAdmin,
		}
		correctEndpointRequest = `
			{
				"name":"gpt2-dep1",
				"modelId":"348967bb-386d-41d0-93cb-ca30f7bbd07d",
				"cpu": 24,
				"memoryInGi": 256,
				"gpu": 1,
				"gpuProduct": "NVIDIA-A100-PCIE-40GB",
				"minInstances": 1,
				"maxInstances": 1,
				"engine": "tgi"
			}`
		correctRequestForCPUMode = `
			{
				"name":"gpt2-dep1",
				"modelId":"348967bb-386d-41d0-93cb-ca30f7bbd07d",
				"cpu": 24,
				"memoryInGi": 256,
				"gpu": 0,
				"minInstances": 1,
				"maxInstances": 1,
				"engine": "vllm"
			}`
		wrongEndpointRequest = `
			{
				"name":"gpt2-dep1",
				"modelId":"348967bb-386d-41d0-93cb-ca30f7bbd07d",
				"cpu": "24",
				"memoryInGi": "256",
				"gpu": "1",
				"engine": "tgi",
			}`
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockCtrl = gomock.NewController(GinkgoT())
		mockEndpointService = mock_service.NewMockIEndpointService(mockCtrl)
		mockAuthService = mock_middleware.NewMockIAuthenticationMiddleware(mockCtrl)
		validateAccessTokenHandler := func(c *gin.Context) {
			c.Next()
		}
		mockAuthService.EXPECT().ValidateAccessToken(auth.AllowAll).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(1)

	})

	Context("Test Create Endpoint Request", func() {
		It("Create Endpoint Successful", func() {
			validContext, router := getContext("v1/endpoints", correctEndpointRequest, "POST")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().Create(userContext, getCreateEndpointRequest()).Return("123", nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Create Endpoint Successful, CPU Mode", func() {
			validContext, router := getContext("v1/endpoints", correctRequestForCPUMode, "POST")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().Create(userContext, getCreateEndpointRequestForCPU()).Return("123", nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Create Endpoint Successful, CPU Mode not supported for NIM", func() {
			createNimCPURequest := `
			{
				"name":"gpt2-dep1",
				"modelId":"348967bb-386d-41d0-93cb-ca30f7bbd07d",
				"cpu": 24,
				"memoryInGi": 256,
				"gpu": 0,
				"minInstances": 1,
				"maxInstances": 1,
				"engine": "nim"
			}`

			validContext, router := getContext("v1/endpoints", createNimCPURequest, "POST")
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create Endpoint Successful, Only NVIDIA-L40S GPU is supported", func() {
			createNimCPURequest := `
			{
				"name":"gpt2-dep1",
				"modelId":"348967bb-386d-41d0-93cb-ca30f7bbd07d",
				"cpu": 24,
				"memoryInGi": 256,
				"gpu": 1,
				"minInstances": 1,
				"maxInstances": 1,
				"engine": "nim",
				"gpuProduct": "NVIDIA-A100-PCIE-40GB"
			}`

			validContext, router := getContext("v1/endpoints", createNimCPURequest, "POST")
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create Endpoint unsuccessful: Create Service gives error", func() {
			validContext, router := getContext("v1/endpoints", correctEndpointRequest, "POST")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().Create(userContext, getCreateEndpointRequest()).Return("", &e.Error{Type: e.GenericError, Msg: "failed to create endpoint"}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Create Endpoint unsuccessful: Binding error", func() {
			validContext, router := getContext("v1/endpoints", wrongEndpointRequest, "POST")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create Endpoint unsuccessful: CreateEndpointRequest dto validation failed", func() {
			validContext, router := getContext("v1/endpoints", "{}", "POST")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Create(validContext)
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})

	Context("Test Get by ID Endpoint Request", func() {
		endpointID := uuid.NewString()
		It("Get by ID Endpoint Successful", func() {
			validContext, router := getContext("v1/endpoints", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			mockEndpointService.EXPECT().GetByID(userContext, endpointID, dto.ExpansionItems{constants.ActualInstances: true}).Return(dto.GetEndpointResponse{ID: endpointID}, nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Get by ID With Status Endpoint Successful", func() {
			validContext, router := getContext("v1/endpoints?expand=status", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			mockEndpointService.EXPECT().GetByID(userContext, endpointID, dto.ExpansionItems{constants.Status: true, constants.ActualInstances: true}).Return(dto.GetEndpointResponse{ID: endpointID}, nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Get by ID With Status Endpoint non status expand fails", func() {
			validContext, router := getContext("v1/endpoints?expand=status&expand=version", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Get by ID Endpoint unsuccessful: Get Service gives error", func() {
			validContext, router := getContext("v1/endpoints", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			mockEndpointService.EXPECT().GetByID(userContext, endpointID, dto.ExpansionItems{constants.ActualInstances: true}).Return(dto.GetEndpointResponse{}, &e.Error{Type: e.DBError, Msg: "failed to get endpoint"}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	Context("Test Delete Endpoint Request", func() {
		endpointID := uuid.NewString()
		It("Delete Endpoint Successful", func() {
			validContext, router := getContext("v1/endpoints?force=true", "", "DELETE")
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().Delete(userContext, endpointID, true).Return(nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Delete Endpoint unsuccessful: Delete Service gives error", func() {
			validContext, router := getContext("v1/endpoints/", "", "DELETE")
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().Delete(userContext, endpointID, false).Return(&e.Error{Type: e.DBError, Msg: "failed to delete endpoint"}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Delete Endpoint unsuccessful: force delete parsing error", func() {
			validContext, router := getContext("v1/endpoints?force=random", "", "DELETE")
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})

	Context("Test List Endpoint Request", func() {
		listOptions := dto.ListOptions{Filters: []dto.FilterOptions{}, Limit: &defaultLimit, Offset: &defaultOffset}
		listOptionsComparator := getListOptionsComparator(listOptions)
		It("List Endpoint Successful", func() {
			expectedResult := []dto.GetEndpointResponse{
				{
					Name:      "endpoint_name",
					ModelName: "endpoint_model_name",
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				{
					Name:      "endpoint_name2",
					ModelName: "endpoint_model_name2",
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
			}
			validContext, router := getContext("v1/endpoints/", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().List(userContext, dto.ExpansionItems{}, gomock.Cond(listOptionsComparator)).Return(expectedResult, int64(2), nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("List Endpoint unsuccessful: error parsing limit", func() {
			validContext, router := getContext("v1/endpoints?limit=a", "", "GET")
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List Endpoint unsuccessful: unsupported query param", func() {
			validContext, router := getContext("v1/endpoints?name=a", "", "GET")
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List Endpoint unsuccessful: List Service gives error", func() {
			validContext, router := getContext("v1/endpoints/", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().List(userContext, dto.ExpansionItems{}, gomock.Cond(listOptionsComparator)).Return([]dto.GetEndpointResponse{}, int64(0), &e.Error{Type: e.DBError, Msg: "failed to list endpoints"}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("List Endpoint unsuccessful: invalid owner from query param", func() {
			validContext, router := getContext("v1/endpoints?owner_id=invalid_owner", "", "GET")
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusForbidden))
		})
	})

	Context("Test ListAPIKeys Request", func() {
		endpointID := uuid.NewString()
		validAPIKey := model.APIKey{
			BaseModel: model.BaseModel{
				ID:        "27ea50bf-25eb-4731-a152-629fe708de5b",
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			},
			UserID:    userID,
			MaskedKey: "e18...7a6",
			Name:      "Valid_API_Key1",
			Status:    true,
			Endpoints: nil,
		}
		It("ListAPIKeys Successful", func() {
			validContext, router := getContext("v1/endpoints/api_keys", "", "GET")
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().ListAPIKeysByEndpoint(userContext, endpointID).Return([]model.APIKey{validAPIKey}, nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.ListAPIKeys(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("ListAPIKeys unsuccessful: ListAPIKeysByEndpoint Service gives error", func() {
			validContext, router := getContext("v1/endpoints/api_keys", "", "GET")
			validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
			validContext.Set(userIDKey, userID)
			validContext.Set(roleKey, role)
			mockEndpointService.EXPECT().ListAPIKeysByEndpoint(userContext, endpointID).Return([]model.APIKey{}, &e.Error{Type: e.DBError, Msg: "failed to list api keys"}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.ListAPIKeys(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	// Context("Test Update Endpoint Request", func() {
	// 	endpointID := uuid.NewString()
	// 	correctRequest := `
	// 		{
	// 			"description": "Update endpoint",
	// 			"minInstances": 1,
	// 			"maxInstances": 1
	// 		}`

	// 	It("Update Endpoint Successful", func() {
	// 		validContext, router := getContext("v1/endpoints/", correctRequest, "PATCH")
	// 		validContext.Set(userIDKey, userID)
	// 		validContext.Set(roleKey, role)
	// 		validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
	// 		mockEndpointService.EXPECT().Update(userContext, endpointID, getUpdateEndpoint()).Return(nil).Times(1)
	// 		testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
	// 		testEndpointController.Update(validContext)
	// 		Expect(validContext.IsAborted()).Should(BeFalse())
	// 		Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
	// 	})

	// 	It("Update Endpoint Unsuccessful: Update Service gives error", func() {
	// 		validContext, router := getContext("v1/endpoints/", correctRequest, "PATCH")
	// 		validContext.Set(userIDKey, userID)
	// 		validContext.Set(roleKey, role)
	// 		validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
	// 		mockEndpointService.EXPECT().Update(userContext, endpointID, getUpdateEndpoint()).Return(&e.Error{Type: e.DBError, Msg: "failed to update Endpoint"}).Times(1)
	// 		testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
	// 		testEndpointController.Update(validContext)
	// 		Expect(validContext.IsAborted()).Should(BeTrue())
	// 		Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
	// 	})

	// 	It("Update Endpoint unsuccessful: Binding error", func() {
	// 		validContext, router := getContext("v1/endpoints/", wrongEndpointRequest, "PATCH")
	// 		validContext.Set(userIDKey, userID)
	// 		validContext.Set(roleKey, role)
	// 		validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
	// 		testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
	// 		testEndpointController.Update(validContext)
	// 		Expect(validContext.IsAborted()).Should(BeTrue())
	// 		Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
	// 	})

	// 	It("Update Endpoint unsuccessful: UpdateEndpointRequest dto validation failed", func() {
	// 		updateEndpointRequest := `
	// 		{
	// 			"minInstances": 1,
	// 			"maxInstances": 2
	// 		}`
	// 		validContext, router := getContext("v1/endpoints/", updateEndpointRequest, "PATCH")
	// 		validContext.Set(userIDKey, userID)
	// 		validContext.Set(roleKey, role)
	// 		validContext.Params = append(validContext.Params, gin.Param{Key: "endpoint_id", Value: endpointID})
	// 		testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
	// 		testEndpointController.Update(validContext)
	// 		Expect(validContext.IsAborted()).Should(BeTrue())
	// 		Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
	// 	})
	// })

	Context("Test Validate Endpoint Name", func() {
		It("Validate Endpoint Name Successful", func() {
			validEndpointName := "valid-endpoint-name"
			validContext, router := getContext("v1/endpoints/validate", "", "POST")
			// set query params
			u := url.Values{}
			u.Add("endpoint_name", validEndpointName)
			validContext.Request.URL.RawQuery = u.Encode()
			mockEndpointService.EXPECT().ValidateEndpointName(validEndpointName).Return(nil).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.ValidateEndpoint(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Validate Endpoint Name Unsuccessful: endpoint name length more than allowed limit", func() {
			invalidEndpointName := "invalid-endpoint-name-length-more-than-allowed-limit"
			validContext, router := getContext("v1/endpoints/validate", "", "POST")
			// set query params
			u := url.Values{}
			u.Add("endpoint_name", invalidEndpointName)
			validContext.Request.URL.RawQuery = u.Encode()
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.ValidateEndpoint(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Validate Endpoint Name Unsuccessful: service layer returns error", func() {
			invalidEndpointName := "1-invalid-name"
			validContext, router := getContext("v1/endpoints/validate", "", "POST")
			// set query params
			u := url.Values{}
			u.Add("endpoint_name", invalidEndpointName)
			validContext.Request.URL.RawQuery = u.Encode()
			mockEndpointService.EXPECT().ValidateEndpointName(invalidEndpointName).Return(&e.Error{Type: e.ValidationError, Msg: fmt.Sprintf("invalid endpoint name: wrong format of string for name %s", invalidEndpointName)}).Times(1)
			testEndpointController := v1.NewEndpointController(router.Group("/v1"), logger, endpointValidator, mockEndpointService, mockAuthService)
			testEndpointController.ValidateEndpoint(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})
})

// func getUpdateEndpoint() dto.UpdateEndpointRequest {
// 	description := "Update endpoint"
// 	instances := int64(1)
// 	return dto.UpdateEndpointRequest{
// 		Description:  &description,
// 		MinInstances: &instances,
// 		MaxInstances: &instances,
// 	}
// }
