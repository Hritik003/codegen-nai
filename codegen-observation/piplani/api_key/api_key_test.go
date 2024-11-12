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

func getCreateKey() dto.APIKeyCreateRequest {
	return dto.APIKeyCreateRequest{
		Name:        "key1",
		EndpointIDs: []string{"bf9e69be-c60f-44ff-9a19-c65146a9c310"},
	}
}

func getUpdateKey() dto.APIKeyUpdateRequest {
	inactiveStatus := string(constants.APIKeyInactive)
	return dto.APIKeyUpdateRequest{
		Status:      &inactiveStatus,
		EndpointIDs: []string{"bf9e69be-c60f-44ff-9a19-c65146a9c310"},
	}
}

var _ = Describe("Test API_KEY Controller", func() {

	var (
		mockCtrl          *gomock.Controller
		mockAPIKeyService *mock_service.MockIAPIKeyService
		mockAuthService   *mock_middleware.MockIAuthenticationMiddleware
		logger            = logger.NewZAPLogger()
		apiKeyValidator   = naivalidator.NewValidator(logger)
		createdAt         = time.Now()
		updatedAt         = time.Now()
		userID            = uuid.NewString()
		mlAdminRole       = "MLAdmin"
		supportedFields   = []string{constants.Name, constants.OwnerID}
		userContext       = dto.UserContext{
			UserID: userID,
			Role:   model.MLAdmin,
		}
		correctCreateKeyRequest = `
			{
				"name": "key1",
				"endpoints": ["bf9e69be-c60f-44ff-9a19-c65146a9c310"]
			}`
		wrongCreateKeyRequest = `
			{
				"name": "key1",
				"endpoints": ["04bf3f32-42ee-4ee8-997e-ad4adabbca6c"],
			}`

		correctUpdateKeyRequest = `
			{
				"status": "inactive",
				"endpoints": ["bf9e69be-c60f-44ff-9a19-c65146a9c310"]
			}`
		wrongUpdateKeyRequest = `
			{
				"status": "inactive"
				"endpoints": ["04bf3f32-42ee-4ee8-997e-ad4adabbca6c"],
			}`
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockCtrl = gomock.NewController(GinkgoT())
		mockAPIKeyService = mock_service.NewMockIAPIKeyService(mockCtrl)
		mockAuthService = mock_middleware.NewMockIAuthenticationMiddleware(mockCtrl)
		validateAccessTokenHandler := func(c *gin.Context) {
			c.Next()
		}
		mockAuthService.EXPECT().ValidateAccessToken(auth.AllowAll).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(1)
	})

	Context("Test Create API Key Request", func() {
		It("Create API Key Successful", func() {
			validContext, router := getContext("v1/apikeys", correctCreateKeyRequest, "POST")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.Name:    {"key1"},
				constants.OwnerID: {userContext.UserID},
			}, supportedFields)
			listOptionsComparator := getListOptionsComparator(listOpts)

			mockAPIKeyService.EXPECT().List(dto.UserContext{}, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{}, int64(0), nil).Times(1)
			mockAPIKeyService.EXPECT().Create(userContext, getCreateKey()).Return(dto.APIKeyCreateResponse{GeneratedKey: "key", ID: "1"}, nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Create API Key Unsuccessful: APIKeyService error", func() {
			validContext, router := getContext("v1/apikeys", correctCreateKeyRequest, "POST")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.Name:    {"key1"},
				constants.OwnerID: {userContext.UserID},
			}, supportedFields)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockAPIKeyService.EXPECT().List(dto.UserContext{}, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{}, int64(0), nil).Times(1)
			mockAPIKeyService.EXPECT().Create(userContext, getCreateKey()).Return(dto.APIKeyCreateResponse{}, &e.Error{Type: e.DBError, Msg: "Failed to create API Key"}).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Create API Key unsuccessful: Binding error", func() {
			validContext, router := getContext("v1/apikeys", wrongCreateKeyRequest, "POST")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create API Key unsuccessful: Uniqueness Constraints error, list error", func() {
			validContext, router := getContext("v1/apikeys", correctCreateKeyRequest, "POST")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.Name:    {"key1"},
				constants.OwnerID: {userContext.UserID},
			}, supportedFields)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockAPIKeyService.EXPECT().List(dto.UserContext{}, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{}, int64(0), &e.Error{Type: e.DBError, Msg: "Failed to list API Key"}).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Create API Key unsuccessful: Uniqueness Constraints error, already exists", func() {
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			testData.Status = false
			validContext, router := getContext("v1/apikeys", correctCreateKeyRequest, "POST")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			var listOpts dto.ListOptions
			listOpts.AddEqualToFiltersFromMap(map[string][]string{
				constants.Name:    {"key1"},
				constants.OwnerID: {userContext.UserID},
			}, supportedFields)
			listOptionsComparator := getListOptionsComparator(listOpts)
			mockAPIKeyService.EXPECT().List(dto.UserContext{}, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{testData}, int64(1), nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Create API Key unsuccessful: create request dto validation failed", func() {
			validContext, router := getContext("v1/apikeys", "{}", "POST")
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Create(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})

	Context("Test List API Key Request", func() {
		defaultLimit := constants.DefaultDBGetLimit
		listOptions := dto.ListOptions{
			Filters: []dto.FilterOptions{},
			Limit:   &defaultLimit,
			Offset:  &defaultOffset,
		}
		listOptionsComparator := getListOptionsComparator(listOptions)

		It("List API Key Successful", func() {
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext, router := getContext("v1/apikeys", "", "GET")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			mockAPIKeyService.EXPECT().List(userContext, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{testData}, int64(1), nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("List API Key Successful, Deactivate key", func() {
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			testData.Status = false
			validContext, router := getContext("v1/apikeys", "", "GET")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			mockAPIKeyService.EXPECT().List(userContext, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{testData}, int64(1), nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("List API Key Unsuccessful: error parsing limit", func() {
			validContext, router := getContext("v1/apikeys?limit=a", "", "GET")
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List API Key Unsuccessful: unsupported filter", func() {
			validContext, router := getContext("v1/apikeys?name=a", "", "GET")
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("List API Key Unsuccessful: APIKeyService error", func() {
			validContext, router := getContext("v1/apikeys", "", "GET")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			mockAPIKeyService.EXPECT().List(userContext, gomock.Cond(listOptionsComparator)).Return([]model.APIKey{}, int64(0), &e.Error{Type: e.DBError, Msg: "Failed to create API Key"}).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("List API Key Unsuccessful: invalid owner from query param", func() {
			validContext, router := getContext("v1/apikeys?owner_id=invalid_owner", "", "GET")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.List(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusForbidden))
		})
	})

	Context("Test Delete API Key Request", func() {
		It("Delete API Key Successful", func() {
			validContext, router := getContext("v1/apikeys", "", "DELETE")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			mockAPIKeyService.EXPECT().Delete(userContext, testData.ID).Return(nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Delete API Key Unsuccessful: Delete Service gives error", func() {
			validContext, router := getContext("v1/apikeys", "", "DELETE")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			mockAPIKeyService.EXPECT().Delete(userContext, testData.ID).Return(&e.Error{Type: e.DBError, Msg: "Failed to create API Key"}).Times(1)
			testEndpointController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testEndpointController.Delete(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})
	})

	Context("Test Update API Key Request", func() {
		It("Update API Key Successful", func() {
			validContext, router := getContext("v1/apikeys", correctUpdateKeyRequest, "PATCH")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			mockAPIKeyService.EXPECT().Update(userContext, testData.ID, getUpdateKey()).Return(nil).Times(1)
			testAPIKeyController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testAPIKeyController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeFalse())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
		})

		It("Update API Key Unsuccessful: Update Service gives error", func() {
			validContext, router := getContext("v1/apikeys", correctUpdateKeyRequest, "PATCH")
			validContext.Set("userID", userID)
			validContext.Set("role", mlAdminRole)
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			mockAPIKeyService.EXPECT().Update(userContext, testData.ID, getUpdateKey()).Return(&e.Error{Type: e.DBError, Msg: "Failed to create API Key"}).Times(1)
			testEndpointController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testEndpointController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
		})

		It("Update API Key Unsuccessful: Binding error", func() {
			validContext, router := getContext("v1/apikeys", wrongUpdateKeyRequest, "PATCH")
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			testEndpointController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testEndpointController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})

		It("Update API Key Unsuccessful: update request dto validation failed", func() {
			updateKeyRequest := `
			{
				"status": "random"
			}`
			validContext, router := getContext("v1/apikeys", updateKeyRequest, "PATCH")
			testData := getValidAPIKeyTestData(createdAt, updatedAt)
			validContext.Params = append(validContext.Params, gin.Param{Key: "apikey_id", Value: testData.ID})
			testEndpointController := v1.NewAPIKeyController(router.Group("/v1"), logger, apiKeyValidator, mockAPIKeyService, mockAuthService)
			testEndpointController.Update(validContext)
			Expect(validContext.IsAborted()).Should(BeTrue())
			Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
		})
	})
})

func getValidAPIKeyTestData(createdAt time.Time, updatedAt time.Time) model.APIKey {
	validAPIKey := model.APIKey{
		BaseModel: model.BaseModel{
			ID:        "1",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		UserID:    uuid.NewString(),
		MaskedKey: "e18...7a6",
		Name:      "key1",
		Status:    true,
		Endpoints: []model.Endpoint{
			{
				BaseModel: model.BaseModel{
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Name: "dep1",
			},
		},
	}

	return validAPIKey
}
