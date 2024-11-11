package v1_test

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	v1 "github.com/nutanix-core/nai-api/iep/api/v1"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/dto"
	auth "github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/model"
	naivalidator "github.com/nutanix-core/nai-api/iep/internal/validator"
	mock_middleware "github.com/nutanix-core/nai-api/iep/mocks/middleware"
	mock_service "github.com/nutanix-core/nai-api/iep/mocks/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ClusterController test", func() {

	Context("Test Cluster Controller", func() {
		var (
			mockCtrl                   *gomock.Controller
			mockClusterService         *mock_service.MockIClusterService
			mockModelService           *mock_service.MockIModelService
			mockDataConsistencyService *mock_service.MockIDataConsistencyService
			mockAuthService            *mock_middleware.MockIAuthenticationMiddleware
			logger                     = logger.NewZAPLogger()
			clusterValidator           = naivalidator.NewValidator(logger)
			userContext                = dto.UserContext{
				UserID:   "system",
				UserName: "system",
				Role:     model.SuperAdmin,
			}
		)

		BeforeEach(func() {
			gin.SetMode(gin.TestMode)
			mockCtrl = gomock.NewController(GinkgoT())
			mockClusterService = mock_service.NewMockIClusterService(mockCtrl)
			mockModelService = mock_service.NewMockIModelService(mockCtrl)
			mockDataConsistencyService = mock_service.NewMockIDataConsistencyService(mockCtrl)
			mockAuthService = mock_middleware.NewMockIAuthenticationMiddleware(mockCtrl)
			validateAccessTokenHandler := func(c *gin.Context) {
				c.Next()
			}
			mockAuthService.EXPECT().ValidateAccessToken(auth.AllowAll).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(2)
			mockAuthService.EXPECT().ValidateAccessToken(auth.AllowSuperAdmin).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(1)
			mockAuthService.EXPECT().ValidateAccessToken(auth.AllowAdmin).Return(gin.HandlerFunc(validateAccessTokenHandler)).Times(1)
		})

		Context("test get clusterinfo", func() {
			It("GetClusterInfo Successful", func() {
				validContext, router := getContext("v1/cluster/info", "", "GET")
				mockClusterService.EXPECT().GetK8SClusterNodesInfo().Return(getClusterInfoResponse(), nil).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterInfo(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("GetClusterInfo unsuccessful: Cluster Service gives error", func() {
				validContext, router := getContext("v1/cluster/info", "", "GET")
				mockClusterService.EXPECT().GetK8SClusterNodesInfo().Return(dto.GetClusterResponse{}, &e.Error{Type: e.K8sError, InternalErr: errors.New("failed to fetch cluster info")}).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterInfo(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})
		})

		Context("test get clusterConfig", func() {
			listOptions := dto.ListOptions{Filters: []dto.FilterOptions{}, Limit: &defaultLimit, Offset: &defaultOffset}
			listOptionsComparator := getListOptionsComparator(listOptions)
			It("GetClusterConfig Successful", func() {
				currentTime := time.Now()
				clusterConfig := dto.ClusterConfig{
					EULA: &dto.EULA{
						Accepted:  false,
						UpdatedAt: &currentTime,
						Name:      "system",
						Company:   "nutanix",
					},
					Pulse: &dto.Pulse{
						Accepted:  true,
						UpdatedAt: &currentTime,
					},
					Language: &dto.Language{
						Name: enum.EnglishLanguage,
					},
				}
				validContext, router := getContext("v1/cluster/config", "", "GET")
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(clusterConfig, nil).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("GetClusterConfig unsuccessful: Unsupported query parameters", func() {
				validContext, router := getContext("v1/cluster/config?invalid_column=random_value", "", "GET")
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("GetClusterConfig returns config with empty eula", func() {
				currentTime := time.Now()
				clusterConfig := dto.ClusterConfig{
					EULA: &dto.EULA{},
					Pulse: &dto.Pulse{
						Accepted:  true,
						UpdatedAt: &currentTime,
					},
					Language: &dto.Language{
						Name: enum.EnglishLanguage,
					},
				}
				validContext, router := getContext("v1/cluster/config", "", "GET")
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(clusterConfig, nil).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("GetClusterConfig unsuccessful: GetConfig service fails", func() {
				validContext, router := getContext("v1/cluster/config", "", "GET")
				configError := &e.Error{Type: e.DBError, InternalErr: errors.New("error getting config")}
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(dto.ClusterConfig{}, configError).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})

			It("GetClusterConfig: GetPulse Successful with default values", func() {
				currentTime := time.Now()
				clusterConfig := dto.ClusterConfig{
					EULA: &dto.EULA{
						Accepted:  false,
						UpdatedAt: &currentTime,
						Name:      "system",
						Company:   "nutanix",
					},
					Pulse: &dto.Pulse{
						Accepted: false,
					},
					Language: &dto.Language{
						Name: enum.EnglishLanguage,
					},
				}
				validContext, router := getContext("v1/cluster/config", "", "GET")
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(clusterConfig, nil).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})
		})

		Context("test update eula", func() {
			updatedAt := time.Now()
			eulaConfig := dto.ClusterConfig{
				EULA: &dto.EULA{
					Accepted:  true,
					UpdatedAt: &updatedAt,
					Name:      "system",
					Company:   "nutanix",
				},
			}
			pulseConfig := dto.ClusterConfig{
				Pulse: &dto.Pulse{
					Accepted:  true,
					UpdatedAt: &updatedAt,
				},
			}
			languageConfig := dto.ClusterConfig{
				Language: &dto.Language{
					Name: enum.EnglishLanguage,
				},
			}
			currentEulaConfig := dto.ClusterConfig{
				EULA: &dto.EULA{
					Accepted:  false,
					UpdatedAt: &updatedAt,
					Name:      "system",
					Company:   "nutanix",
				},
			}
			correctUpdateEULARequest := `
			{
				"eula": {
					"accepted": true,
					"name": "system",
					"company": "nutanix"
				}
			}`
			correctUpdatePulseRequest := `
			{
				"pulse": {
					"accepted": true
				}
			}`
			correctUpdateLanguageRequest := `
			{
				"languageConfig": {
					"language": "English"
				}
			}`
			wrongUpdateLanguageRequest := `
			{
				"languageConfig": {
					"language": "NotSupportedLanguage"
				}
			}`
			pulseComparator := func(x any) bool {
				argPulse := x.(dto.ClusterConfig).Pulse
				return argPulse.Accepted == pulseConfig.Pulse.Accepted
			}
			languageComparator := func(x any) bool {
				argLanguage := x.(dto.ClusterConfig).Language
				return argLanguage.Name == languageConfig.Language.Name
			}
			eulaComparator := func(x any) bool {
				argEula := x.(dto.ClusterConfig).EULA
				return argEula.Accepted == eulaConfig.EULA.Accepted && argEula.Name == eulaConfig.EULA.Name && argEula.Company == eulaConfig.EULA.Company
			}
			var listOptions dto.ListOptions
			listOptions.AddEqualToFiltersFromMap(map[string][]string{
				"type": {string(enum.ConfigTypeEULA)},
			}, []string{"type"})
			listOptionsComparator := getListOptionsComparator(listOptions)

			It("UpdateClusterConfig Successful: Accept EULA", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdateEULARequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(currentEulaConfig, nil)
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(eulaComparator)).Return(nil)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("UpdateClusterConfig fails: EULA already accepted", func() {
				acceptedEULA := dto.ClusterConfig{
					EULA: &dto.EULA{
						Accepted:  true,
						UpdatedAt: &updatedAt,
						Name:      "system",
						Company:   "nutanix",
					},
				}
				validContext, router := getContext("v1/cluster/config", correctUpdateEULARequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(acceptedEULA, nil)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("UpdateClusterConfig fails: Get current config fails", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdateEULARequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				err := &e.Error{Type: e.DBError}
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(dto.ClusterConfig{}, err)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})

			It("UpdateClusterConfig fails: update config service error", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdateEULARequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				eulaError := &e.Error{Type: e.DBError, InternalErr: errors.New("error udpating eula")}
				mockClusterService.EXPECT().GetConfig(gomock.Cond(listOptionsComparator)).Return(currentEulaConfig, nil)
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(eulaComparator)).Return(eulaError)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})

			It("UpdateClusterConfig fails: EULA validation error", func() {
				eulaAcceptFalseRequest := `
				{
					"eula": {
						"accepted": false
					}
				}`
				validContext, router := getContext("v1/cluster/config", eulaAcceptFalseRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("UpdateClusterConfig fails: binding error", func() {
				bindErrorRequest := `{
					"eula": {
						"accepted":
					}
				}`
				validContext, router := getContext("v1/cluster/config", bindErrorRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("UpdateClusterConfig Successful: update pulse", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdatePulseRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(pulseComparator)).Return(nil)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("UpdateClusterConfig fails: update pulse service error", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdatePulseRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				pulseError := &e.Error{Type: e.DBError, InternalErr: errors.New("error udpating pulse")}
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(pulseComparator)).Return(pulseError)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})

			It("UpdateClusterConfig Successful: update language", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdateLanguageRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(languageComparator)).Return(nil)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("UpdateClusterConfig fails: update language service error", func() {
				validContext, router := getContext("v1/cluster/config", correctUpdateLanguageRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				languageError := &e.Error{Type: e.DBError, InternalErr: errors.New("error udpating language")}
				mockClusterService.EXPECT().UpdateConfig(userContext, gomock.Cond(languageComparator)).Return(languageError)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})

			It("UpdateClusterConfig fails: update language validation error", func() {
				validContext, router := getContext("v1/cluster/config", wrongUpdateLanguageRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("UpdateClusterConfig fails: all config nil validation error", func() {
				validContext, router := getContext("v1/cluster/config", "{}", "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})

			It("UpdateClusterConfig fails: validation error", func() {
				wrongUpdateRequest := `{}`
				validContext, router := getContext("v1/cluster/config", wrongUpdateRequest, "PATCH")
				validContext.Set("userID", userContext.UserID)
				validContext.Set("userName", userContext.UserName)
				validContext.Set("role", string(userContext.Role))
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.UpdateClusterConfig(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusBadRequest))
			})
		})

		Context("test get cluster health", func() {
			It("GetClusterHealth Successful", func() {
				validContext, router := getContext("v1/cluster/health", "", "GET")
				testData := dto.InconsistentResource{
					ResourceType: dto.EndpointResource,
					ID:           "llama3",
					Name:         "llama3",
					Description:  "Endpoint not found in k8s",
				}
				mockDataConsistencyService.EXPECT().GetInconsistentData().Return([]dto.InconsistentResource{testData}, nil).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterHealth(validContext)
				Expect(validContext.IsAborted()).Should(BeFalse())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusOK))
			})

			It("GetClusterHealth unsuccessful: data consistency service gives error", func() {
				validContext, router := getContext("v1/cluster/health", "", "GET")
				mockDataConsistencyService.EXPECT().GetInconsistentData().Return([]dto.InconsistentResource{}, &e.Error{Type: e.DBError, Msg: "Failed to list endpoints"}).Times(1)
				testClusterController := v1.NewClusterController(router.Group("/v1"), logger, clusterValidator, mockClusterService, mockModelService, mockDataConsistencyService, mockAuthService)
				testClusterController.GetClusterHealth(validContext)
				Expect(validContext.IsAborted()).Should(BeTrue())
				Expect(validContext.Writer.Status()).Should(Equal(http.StatusInternalServerError))
			})
		})
	})
})

func getClusterInfoResponse() dto.GetClusterResponse {
	return dto.GetClusterResponse{
		Version:        "v1.25.6",
		Status:         "online",
		Gpu:            0,
		GpuPassthrough: "true",
		CPU:            4,
		Memory:         8,
		Disk:           8,
		Nodes: []dto.GetClusterNodeResponse{
			{
				Name:       "node1",
				IP:         "192.168.1.1",
				Status:     "online",
				Version:    "v1.25.6",
				Gpu:        0,
				GpuMachine: "",
				GpuProduct: "",
				CPU:        4,
				Memory:     8,
				Disk:       8,
			},
		},
	}
}
