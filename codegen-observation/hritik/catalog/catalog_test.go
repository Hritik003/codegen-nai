package v1_test

import (
	"net/http"
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
	naivalidator "github.com/nutanix-core/nai-api/iep/internal/validator"
	mock_middleware "github.com/nutanix-core/nai-api/iep/mocks/middleware"
	mock_service "github.com/nutanix-core/nai-api/iep/mocks/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var (
	tokenRequired = true
	modelWeights  = float64(30)
	activations   = float64(1)
	kvCache       = float64(0)
	contextLength = int64(4096)
	modelSizeInGB = int64(20)
	cpu           = int64(6)
	ram           = float64(24)
)

func getCreateCatalog() dto.CreateCatalogRequest {
	return dto.CreateCatalogRequest{
		ModelName:     "mistralai/Mistral-7B-Instruct-v0.2",
		ModelRevision: "1234",
		Description:   "mistral model",
		ModelType:     enum.TextGeneration,
		SourceHub:     enum.HFSourceHub,
		ContextLength: &contextLength,
		TokenRequired: &tokenRequired,
		ModelSizeInGB: &modelSizeInGB,
		License:       "License to use llama2",
		Developer:     "Meta",
		ModelURL:      "https://hf.co/meta-llama/Llama-2-7b-chat-hf",
		Quantization:  enum.Float16,
		Runtimes: []dto.CreateRuntimeRequest{
			{
				Name:  enum.TGIEngine,
				Image: "hf:2.0",
				Resources: dto.MinResources{
					CPU: &cpu,
					RAM: &ram,
					GPUMemory: &dto.GPUMemory{
						ModelWeights:        &modelWeights,
						ActivationsPerToken: &activations,
						KVCachePerToken:     &kvCache,
					},
				},
			},
			{
				Name:  enum.VLLMEngine,
				Image: "vllm:1.0",
				Resources: dto.MinResources{
					CPU: &cpu,
					RAM: &ram,
					GPUMemory: &dto.GPUMemory{
						ModelWeights:        &modelWeights,
						ActivationsPerToken: &activations,
						KVCachePerToken:     &kvCache,
					},
				},
			},
		},
	}
}

func getUpdateCatalog() dto.UpdateCatalogRequest {
	tokReq := false
	return dto.UpdateCatalogRequest{
		TokenRequired: &tokReq,
	}
}

func getCatalogRequirementsData() (dto.CatalogRequirements, dto.CatalogRequirementsResponse) {
	gpuMemory := float64(16)
	modelSizeInGB := int64(20)
	modelName := "mistralai/Mistral-7B-Instruct-v0.2"
	catalogRequirements := dto.CatalogRequirements{
		ModelName: modelName,
		Engine:    enum.VLLMEngine,
		GPUMemory: &gpuMemory,
	}
	catalogRequirementsResponse := dto.CatalogRequirementsResponse{
		ModelName:     modelName,
		ModelRevision: "1234",
		ModelType:     "Text Generation",
		Engine:        enum.VLLMEngine,
		SourceHub:     enum.HFSourceHub,
		ModelSizeInGB: modelSizeInGB,
		ResourceTable: []dto.CatalogRequirementsRows{
			{
				GPUCount:      0,
				ContextLength: contextLength,
				CPU:           cpu,
				RAM:           32,
			},
			{
				GPUCount:      2,
				ContextLength: contextLength,
				CPU:           cpu,
				RAM:           48,
			},
		},
	}
	return catalogRequirements, catalogRequirementsResponse
}

var _ = Describe("Test Catalog API Controller", func() {

	var (
		mockCtrl             *gomock.Controller
		mockCatalogService   *mock_service.MockICatalogService
		mockAuthService      *mock_middleware.MockIAuthenticationMiddleware
		logger               = logger.NewZAPLogger()
		catalogValidator     = naivalidator.NewValidator(logger)
		createdAt            = time.Now()
		updatedAt            = time.Now()
		catalogID            = uuid.NewString()
		supportedQueryParams = []string{constants.ModelName, constants.ModelRevision, constants.CreatedBy, "deprecated"}

		correctCreateCatalogRequest = `
			{
				"modelName": "mistralai/Mistral-7B-Instruct-v0.2",
                "modelRevision": "1234",
                "modelType": "Text Generation",
                "sourceHub": "HuggingFace",
                "description": "mistral model",
                "contextLength": 4096,
                "tokenRequired": true,
                "modelSizeInGB": 20,
                "license": "License to use llama2",
                "developer": "Meta",
                "modelUrl": "https://hf.co/meta-llama/Llama-2-7b-chat-hf",
				"quantization": "float16",
                "runtimes": [
                    {
                        "name": "tgi",
						"image": "hf:2.0",
                        "supportedContextLength": 4096,
                        "resources": {
                            "cpu": 6,
                            "ram": 24,
                            "gpuMemory": {
                                "modelWeights": 30,
                                "activationsPerToken": 1,
                                "kvCachePerToken": 0
                            }
                        }
                    },
					{
                        "name": "vllm",
						"image": "vllm:1.0",
                        "supportedContextLength": 4096,
                        "resources": {
                            "cpu": 6,
                            "ram": 24,
                            "gpuMemory": {
                                "modelWeights": 30,
                                "activationsPerToken": 1,
                                "kvCachePerToken": 0
                            }
                        }
                    }
                ]
			}`
		wrongCreateCatalogRequest = `
			{
				"modelName": "mistralai/Mistral-7B-Instruct-v0.2",
				"description": "mistral model",
				"modelType": "Text Generation",
				"contextLength": 4096,
				"tokenRequired": true,
				"resources": {
					"cpu": 6,
					"memory": 24,
					"gpuMemory": 30
					"gpuMemoryOnScale": 4
				},
			}`
		correctUpdateCatalogRequest = `
			{
				"tokenRequired": false
			}`
		wrongUpdateCatalogRequest = `
			{
				"tokenRequired": false,
			}`
		correctCatalogRequirementsRequest = `
			{
				"modelName": "mistralai/Mistral-7B-Instruct-v0.2",
				"engine": "vllm",
				"gpuMemory": 16
			}`
		defaultCatalogRequirementsRequest = `
			{
				"modelName": "mistralai/Mistral-7B-Instruct-v0.2",
				"engine": "vllm"
			}`
		validateErrCatalogRequirementsRequest = `
			{
				"modelName": "mistralai/Mistral-7B-Instruct-v0.2",
				"engine": "invalidEngine",
				"gpuMemory": 16
			}`
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockCtrl = gomock.NewController(GinkgoT())
		mockCatalogService = mock_service.NewMockICatalogService(mockCtrl)
		mockAuthService = mock_middleware.NewMockIAuthenticationMiddleware(mockCtrl)
		validateAccessTokenHandler := func(c *gin.Context) {
			c.Next()
		}
		mockAuthService.EXPECT().ValidateAccessToken(auth.AllowSuperAdmin).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(2)
		mockAuthService.EXPECT().ValidateAccessToken(auth.AllowAll).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(3)
	})

	Context("Test Create Catalog Request", func() {
		catalogEntry := getCreateCatalog()
		It("Create Catalog Successful", func() {
			validContext, router := getContext("v1/catalogs", correctCreateCatalogRequest, "POST")
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.ModelName:     {catalogEntry.ModelName},
				constants.ModelRevision: {catalogEntry.ModelRevision},
				constants.CreatedBy:     {constants.SuperAdmin},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{}, int64(0), nil).Times(1)
			mockCatalogService.EXPECT().Create(catalogEntry).Return("123", nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Create Catalog Unsuccessful: Create Service gives error", func() {
			validContext, router := getContext("v1/catalogs", correctCreateCatalogRequest, "POST")
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.ModelName:     {catalogEntry.ModelName},
				constants.ModelRevision: {catalogEntry.ModelRevision},
				constants.CreatedBy:     {constants.SuperAdmin},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{}, int64(0), nil).Times(1)
			mockCatalogService.EXPECT().Create(getCreateCatalog()).Return("", &e.Error{Type: e.DBError, Msg: "failed to create catalog"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Create Catalog unsuccessful: Binding error", func() {
			validContext, router := getContext("v1/catalogs", wrongCreateCatalogRequest, "POST")
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create Catalog unsuccessful: CreateCatalogRequest dto validation failed", func() {
			validContext, router := getContext("v1/catalogs", "{}", "POST")
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create Catalog unsuccessful: Uniqueness Constraints error, list error", func() {
			validContext, router := getContext("v1/catalogs", correctCreateCatalogRequest, "POST")
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.ModelName:     {catalogEntry.ModelName},
				constants.ModelRevision: {catalogEntry.ModelRevision},
				constants.CreatedBy:     {constants.SuperAdmin},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{}, int64(0), &e.Error{Type: e.DBError, Msg: "Failed to list Catalog"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Create Catalog unsuccessful: Uniqueness Constraints error, already exists", func() {
			mistralEntry := getMistralCatalogEntry(createdAt, updatedAt)
			validContext, router := getContext("v1/catalogs", correctCreateCatalogRequest, "POST")
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.ModelName:     {catalogEntry.ModelName},
				constants.ModelRevision: {catalogEntry.ModelRevision},
				constants.CreatedBy:     {constants.SuperAdmin},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{mistralEntry}, int64(1), nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})

	Context("Test Get by ID Catalog Request", func() {
		It("Get Catalog Successful", func() {
			validContext, router := getContext("v1/catalogs/", "", "GET")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().GetByID(catalogID).Return(getMistralCatalogEntry(createdAt, updatedAt), nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Get Catalog Unsuccessful: GetByID Service gives error", func() {
			validContext, router := getContext("v1/catalogs/", "", "GET")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().GetByID(catalogID).Return(model.Catalog{}, &e.Error{Type: e.DBError, Msg: "failed to get Catalog"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetByID(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	Context("Test List Catalog Request", func() {
		listOptions := dto.ListOptions{Limit: &defaultLimit, Offset: &defaultOffset}
		listOptions.AddEqualToFiltersFromMap(map[string][]string{
			"model_name": {"meta-llama/Llama-2-7b-chat-hf"},
			"deprecated": {"false"},
		}, supportedQueryParams)
		listOptionsComparator := getListOptionsComparator(listOptions)
		It("List Catalog Successful", func() {
			llamaEntry := getLlamaCatalogEntry(createdAt, updatedAt)
			mistralEntry := getMistralCatalogEntry(createdAt, updatedAt)
			validContext, router := getContext("v1/catalogs?model_name=meta-llama/Llama-2-7b-chat-hf", "", "GET")
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{llamaEntry, mistralEntry}, int64(2), nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("List Catalog Unsuccessful: List Service gives error", func() {
			validContext, router := getContext("v1/catalogs?model_name=meta-llama/Llama-2-7b-chat-hf", "", "GET")
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{}, int64(0), &e.Error{Type: e.DBError, Msg: "failed to get Catalogs"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("List Catalog Unsuccessful: Unsupported query parameters", func() {
			validContext, router := getContext("v1/catalogs?name=meta-llama/Llama-2-7b-chat-hf", "", "GET")
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List Catalog Unsuccessful: error parsing limit", func() {
			validContext, router := getContext("v1/catalogs", "", "GET")
			validContext.Request.URL.RawQuery = "limit=a"
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List Catalog non deprecated Successful", func() {
			nonDeprecatedListOptions := dto.ListOptions{Limit: &defaultLimit, Offset: &defaultOffset}
			nonDeprecatedListOptions.AddEqualToFiltersFromMap(map[string][]string{
				"deprecated": {"false"},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(nonDeprecatedListOptions)
			llamaEntry := getLlamaCatalogEntry(createdAt, updatedAt)
			mistralEntry := getMistralCatalogEntry(createdAt, updatedAt)
			validContext, router := getContext("v1/catalogs?deprecated=false", "", "GET")
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{llamaEntry, mistralEntry}, int64(2), nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("List Catalog non deprecated Successful even when list filters is empty", func() {
			nonDeprecatedListOptions := dto.ListOptions{Limit: &defaultLimit, Offset: &defaultOffset}
			nonDeprecatedListOptions.AddEqualToFiltersFromMap(map[string][]string{
				"deprecated": {"false"},
			}, supportedQueryParams)
			listOptionsComparator := getListOptionsComparator(nonDeprecatedListOptions)
			llamaEntry := getLlamaCatalogEntry(createdAt, updatedAt)
			mistralEntry := getMistralCatalogEntry(createdAt, updatedAt)
			validContext, router := getContext("v1/catalogs", "", "GET")
			mockCatalogService.EXPECT().List(gomock.Cond(listOptionsComparator)).Return([]model.Catalog{llamaEntry, mistralEntry}, int64(2), nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

	})

	Context("Test Update Catalog Request", func() {
		It("Update Catalog Successful", func() {
			validContext, router := getContext("v1/catalogs", correctUpdateCatalogRequest, "PATCH")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().Update(catalogID, getUpdateCatalog()).Return(nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Update Catalog Unsuccessful: Update Service gives error", func() {
			validContext, router := getContext("v1/catalogs", correctUpdateCatalogRequest, "PATCH")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().Update(catalogID, getUpdateCatalog()).Return(&e.Error{Type: e.DBError, Msg: "failed to update Catalog"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Update Catalog unsuccessful: Binding error", func() {
			validContext, router := getContext("v1/catalogs", wrongUpdateCatalogRequest, "PATCH")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Update Catalog unsuccessful: UpdateCatalogRequest dto validation failed", func() {
			updateCatalogRequest := `
			{
				"modelSizeInGB": -1
			}`
			validContext, router := getContext("v1/catalogs", updateCatalogRequest, "PATCH")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})

	Context("Test Delete by ID Catalog Request", func() {

		It("Delete Catalog Successful", func() {
			validContext, router := getContext("v1/catalogs/", "", "DELETE")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().Delete(catalogID).Return(nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Delete Catalog Unsuccessful: Delete Service gives error", func() {
			validContext, router := getContext("v1/catalogs/", "", "DELETE")
			validContext.Params = append(validContext.Params, gin.Param{Key: "catalog_id", Value: catalogID})
			mockCatalogService.EXPECT().Delete(catalogID).Return(&e.Error{Type: e.DBError, Msg: "failed to delete Catalog"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	Context("Test GetRequirements Catalog Request", func() {
		catalogReq, catalogReqResponse := getCatalogRequirementsData()
		It("GetRequirements Catalog Successful", func() {
			validContext, router := getContext("v1/catalogs", correctCatalogRequirementsRequest, "POST")
			mockCatalogService.EXPECT().GetRequirements(catalogReq).Return(catalogReqResponse, nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetRequirements(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("GetRequirements Catalog Successful: SetDefaults case", func() {
			setDefaultsCatalogReq := catalogReq
			zeroValue := float64(0)
			setDefaultsCatalogReq.GPUMemory = &zeroValue
			validContext, router := getContext("v1/catalogs", defaultCatalogRequirementsRequest, "POST")
			mockCatalogService.EXPECT().GetRequirements(setDefaultsCatalogReq).Return(catalogReqResponse, nil).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetRequirements(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("GetRequirements Catalog Unsuccessful: Binding Error", func() {
			validContext, router := getContext("v1/catalogs", "{invalid_json}", "POST")
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetRequirements(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("GetRequirements Catalog Unsuccessful: CatalogRequirementsRequest dto validation failed", func() {
			validContext, router := getContext("v1/catalogs", validateErrCatalogRequirementsRequest, "POST")
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetRequirements(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("GetRequirements Catalog Unsuccessful: Service layer error", func() {
			validContext, router := getContext("v1/catalogs", correctCatalogRequirementsRequest, "POST")
			mockCatalogService.EXPECT().GetRequirements(catalogReq).Return(catalogReqResponse, &e.Error{Type: e.DBError, Msg: "failed to get catalog requirements"}).Times(1)
			testCatalogController := v1.NewCatalogController(router.Group("/v1"), logger, catalogValidator, mockCatalogService, mockAuthService)
			testCatalogController.GetRequirements(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})
})

func getMistralCatalogEntry(createdAt time.Time, updatedAt time.Time) model.Catalog {
	catalogID := "2"
	return model.Catalog{
		BaseModel: model.BaseModel{
			ID:        catalogID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ModelName:             "mistralai/Mistral-7B-Instruct-v0.2",
		Description:           "mistral model",
		ModelType:             enum.TextGeneration,
		SourceHub:             enum.HFSourceHub,
		ContextLength:         4096,
		CreatedBy:             "NAI",
		CreatedFromNAIVersion: "1.0",
		TokenRequired:         true,
		Runtimes: []model.Runtime{
			{
				CatalogID: catalogID,
				Name:      enum.TGIEngine,
				Image:     "abc:1",
				MinResources: model.MinResources{
					CPU: 6,
					RAM: 24,
				},
			},
		},
	}
}

func getLlamaCatalogEntry(createdAt time.Time, updatedAt time.Time) model.Catalog {
	catalogID := "1"
	return model.Catalog{
		BaseModel: model.BaseModel{
			ID:        catalogID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ModelName:             "meta-llama/Llama-2-7b-chat-hf",
		Description:           "mistral model",
		ModelType:             enum.TextGeneration,
		SourceHub:             enum.HFSourceHub,
		ContextLength:         4096,
		CreatedBy:             "NAI",
		CreatedFromNAIVersion: "1.0",
		TokenRequired:         true,
		Runtimes: []model.Runtime{
			{
				CatalogID: catalogID,
				Name:      enum.TGIEngine,
				Image:     "abc:1",
				MinResources: model.MinResources{
					CPU: 6,
					RAM: 24,
				},
			},
		},
	}
}
