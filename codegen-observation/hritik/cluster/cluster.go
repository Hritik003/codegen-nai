package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	"github.com/nutanix-core/nai-api/common/response"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/constants/enum"
	"github.com/nutanix-core/nai-api/iep/internal/dto"
	auth "github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	view "github.com/nutanix-core/nai-api/iep/internal/view"
)

// ClusterController struct
type ClusterController struct {
	v1Route                *gin.RouterGroup
	logger                 logger.Logger
	validator              *validator.Validate
	clusterService         service.IClusterService
	dataConsistencyService service.IDataConsistencyService
	modelService           service.IModelService
	authMiddleware         auth.IAuthenticationMiddleware
}

// NewClusterController creates and initiates the route for accessing cluster information
func NewClusterController(v1Route *gin.RouterGroup, logger logger.Logger, validator *validator.Validate, clusterService service.IClusterService, modelService service.IModelService, dataConsistencyService service.IDataConsistencyService, authMiddleware auth.IAuthenticationMiddleware) *ClusterController {
	controller := &ClusterController{v1Route: v1Route, logger: logger, validator: validator, authMiddleware: authMiddleware, clusterService: clusterService, modelService: modelService, dataConsistencyService: dataConsistencyService}
	controller.route()
	return controller
}

// Route requests to the correct function for accessing cluster information
func (cc *ClusterController) route() {
	route := cc.v1Route.Group("/cluster")
	route.GET("/info", cc.authMiddleware.ValidateAccessToken(auth.AllowAll), cc.GetClusterInfo)

	// APIs corresponding to cluster configs. Allow everyone to access but only super admins to modify
	route.GET("/config", cc.authMiddleware.ValidateAccessToken(auth.AllowAll), cc.GetClusterConfig)
	route.PATCH("/config", cc.authMiddleware.ValidateAccessToken(auth.AllowSuperAdmin), cc.UpdateClusterConfig)

	// API to get cluster health data
	route.GET("/health", cc.authMiddleware.ValidateAccessToken(auth.AllowAdmin), cc.GetClusterHealth)
}

// GetClusterInfo godoc
//
//	@Summary		getClusterInfo
//	@Description	retrieves information about the kubernetes cluster
//	@Tags			cluster
//	@Produce		json
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ClusterInfo}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/cluster/info [get]
func (cc *ClusterController) GetClusterInfo(c *gin.Context) {
	succMsg := "Cluster info fetched successfully"
	clusterInfo, err := cc.clusterService.GetK8SClusterNodesInfo()
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Data: view.GetClusterInfo(clusterInfo), Err: err})
}

// GetClusterHealth godoc
//
//	@Summary		getClusterHealth
//	@Description	get cluster health
//	@Tags			cluster
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ClusterHealth}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/cluster/health [get]
func (cc *ClusterController) GetClusterHealth(c *gin.Context) {
	succMsg := "Cluster health fetched successfully"
	inconsistentData, err := cc.dataConsistencyService.GetInconsistentData()
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: err, Data: view.GetClusterHealth(inconsistentData)})
}

// GetClusterConfig godoc
//
//	@Summary		getClusterConfig
//	@Description	retrieves information about the Cluster Configurations
//	@Tags			cluster
//	@Produce		json
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ClusterConfig}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/cluster/config [get]
func (cc *ClusterController) GetClusterConfig(c *gin.Context) {
	succMsg := "Cluster Config fetched successfully"
	errMsg := "failed to get cluster configs"
	supportedQueryParams := []string{constants.Type}
	listOptions, _, err := GetQueryOptionsFromCtx(c, supportedQueryParams)
	if err != nil {
		err := &e.Error{Type: err.Type, InternalErr: err.InternalErr, Msg: fmt.Sprintf("%s: %s", errMsg, err.Msg), Log: err.Log}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: err})
		return
	}

	clusterConfig, err := cc.clusterService.GetConfig(listOptions)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Data: view.GetClusterConfig(clusterConfig), Err: err})
}

// UpdateClusterConfig godoc
//
//	@Summary		UpdateClusterConfig
//	@Description	update cluster config status
//	@Tags			cluster
//	@Accept			json
//	@Produce		json
//	@Param			ClusterConfigUpdateRequest	body		dto.ClusterConfigUpdateRequest				true	"new cluster config update request object"
//	@Param			Authorization				header		string										true	"access token sent via headers"
//	@Success		200							{object}	response.HTTPSuccessWithDataResponseModel	"success response"
//	@Failure		400							{object}	response.HTTPFailureResponseModel			"bad request response"
//	@Failure		401							{object}	response.HTTPFailureResponseModel			"unauthorized response"
//	@Failure		403							{object}	response.HTTPFailureResponseModel			"forbidden response"
//	@Failure		500							{object}	response.HTTPFailureResponseModel			"internal server error response"
//	@Router			/v1/cluster/config [patch]
func (cc *ClusterController) UpdateClusterConfig(c *gin.Context) {
	errMsg := "Failed to update cluster config"
	succMsg := "Cluster config info updated successfully"

	var clusterConfigUpdateRequest dto.ClusterConfigUpdateRequest
	if err := c.ShouldBindJSON(&clusterConfigUpdateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}
	if err := cc.validator.Struct(clusterConfigUpdateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}

	userContext := getUserContext(c)

	var clusterConfig dto.ClusterConfig
	if clusterConfigUpdateRequest.EULA != nil {
		err := cc.buildEULAConfig(*clusterConfigUpdateRequest.EULA, &clusterConfig)
		if err != nil {
			response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: err})
			return
		}
	}
	if clusterConfigUpdateRequest.Pulse != nil {
		cc.buildPulseConfig(*clusterConfigUpdateRequest.Pulse, &clusterConfig)
	}
	if clusterConfigUpdateRequest.Language != nil {
		cc.buildLanguageConfig(*clusterConfigUpdateRequest.Language, &clusterConfig)
	}

	err := cc.clusterService.UpdateConfig(userContext, clusterConfig)

	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: err})
}

func (cc *ClusterController) buildEULAConfig(eulaUpdateRequest dto.EULAAcceptRequest, clusterConfig *dto.ClusterConfig) *e.Error {
	var listOptions dto.ListOptions
	supportedQueryParams := []string{"type"}
	listOptions.AddEqualToFiltersFromMap(map[string][]string{
		constants.Type: {string(enum.ConfigTypeEULA)},
	}, supportedQueryParams)
	currentConfig, err := cc.clusterService.GetConfig(listOptions)
	if err != nil {
		return err
	}

	if currentConfig.EULA.Accepted {
		// if the eula is currently accepted, do not let it be updated
		msg := "Cannot update already accepted eula"
		return &e.Error{Type: e.InvalidValueError, Msg: msg, Log: msg}
	}

	currentTime := time.Now()
	clusterConfig.EULA = &dto.EULA{
		Accepted:  *eulaUpdateRequest.Accepted,
		UpdatedAt: &currentTime,
		Name:      eulaUpdateRequest.Name,
		Company:   eulaUpdateRequest.Company,
	}

	return nil
}

func (cc *ClusterController) buildPulseConfig(pulseUpdateRequest dto.PulseUpdateRequest, clusterConfig *dto.ClusterConfig) {
	currentTime := time.Now()
	clusterConfig.Pulse = &dto.Pulse{
		Accepted:  *pulseUpdateRequest.Accepted,
		UpdatedAt: &currentTime,
	}
}

func (cc *ClusterController) buildLanguageConfig(languageUpdateRequest dto.LanguageUpdateRequest, clusterConfig *dto.ClusterConfig) {
	clusterConfig.Language = &dto.Language{
		Name: languageUpdateRequest.Name,
	}
}
