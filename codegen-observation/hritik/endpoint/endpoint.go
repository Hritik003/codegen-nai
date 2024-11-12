package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	"github.com/nutanix-core/nai-api/common/response"
	"github.com/nutanix-core/nai-api/iep/constants"
	"github.com/nutanix-core/nai-api/iep/internal/dto"
	auth "github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	"github.com/nutanix-core/nai-api/iep/internal/view"
)

// EndpointController struct
type EndpointController struct {
	v1Route         *gin.RouterGroup
	logger          logger.Logger
	validator       *validator.Validate
	endpointService service.IEndpointService
	authMiddleware  auth.IAuthenticationMiddleware
}

// NewEndpointController creates and initiates the route
func NewEndpointController(v1Route *gin.RouterGroup, logger logger.Logger, validator *validator.Validate, endpointService service.IEndpointService, authMiddleware auth.IAuthenticationMiddleware) *EndpointController {
	controller := &EndpointController{v1Route: v1Route, logger: logger, validator: validator, endpointService: endpointService, authMiddleware: authMiddleware}
	controller.route()
	return controller
}

// Route endpoint requests to the correct function
func (ec *EndpointController) route() {
	route := ec.v1Route.Group("/endpoints", ec.authMiddleware.ValidateAccessToken(auth.AllowAll))
	route.POST("", ec.Create)
	route.GET("", ec.List)
	route.GET("/:endpoint_id", ec.GetByID)
	route.GET("/apikeys/:endpoint_id", ec.ListAPIKeys)
	route.DELETE("/:endpoint_id", ec.Delete)
	// route.PATCH("/:endpoint_id", ec.Update) Update Endpoint is currently parked and not supported
	route.POST("/validate", ec.ValidateEndpoint)
}

// Create godoc
//
//	@Summary		create
//	@Description	create a new endpoint
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			endpoint		body		dto.CreateEndpointRequest								true	"new create endpoint request object"
//	@Param			Authorization	header		string													true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ID}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel						"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel						"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel						"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel						"internal server error response"
//	@Router			/v1/endpoints [post]
func (ec *EndpointController) Create(c *gin.Context) {
	var endpoint dto.CreateEndpointRequest
	errMsg := "Failed to create new endpoint"
	succMsg := "Endpoint creation triggered successfully"
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": errMsg, "error": err.Error()})
		return
	}
	endpoint.SetDefaults()
	if err := ec.validator.Struct(endpoint); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}
	userContext := getUserContext(c)
	id, err := ec.endpointService.Create(userContext, endpoint)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err, Data: view.GetID(id)})
}

// GetByID godoc
//
//	@Summary		getByID
//	@Description	get an endpoint by id
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			endpoint_id		path		string															true	"endpoint id"
//	@Param			Authorization	header		string															true	"access token sent via headers"
//	@Param			expand			query		[]string														false	"query param to denote what all extra fields to fetch"	collectionFormat(multi)
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.Endpoint}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel								"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel								"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel								"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel								"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel								"internal server error response"
//	@Router			/v1/endpoints/{endpoint_id} [get]
func (ec *EndpointController) GetByID(c *gin.Context) {
	endpointID := c.Param("endpoint_id")
	succMsg := "Endpoint fetched successfully"
	errMsg := "Failed to get endpoint by id"
	supportedQueryParams := []string{}
	_, expandParams, err := GetQueryOptionsFromCtx(c, supportedQueryParams)
	if err != nil {
		err := &e.Error{Type: err.Type, InternalErr: err.InternalErr, Msg: fmt.Sprintf("%s: %s", errMsg, err.Msg), Log: err.Log}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: err})
		return
	}
	userContext := getUserContext(c)
	expandParams[constants.ActualInstances] = true
	endpoint, err := ec.endpointService.GetByID(userContext, endpointID, expandParams)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err, Data: view.GetEndpoint(endpoint)})
}

// Delete godoc
//
//	@Summary		delete
//	@Description	delete endpoint by id
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			endpoint_id		path		string								true	"endpoint id"
//	@Param			force			query		bool								false	"force delete"
//	@Param			Authorization	header		string								true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessResponseModel	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel	"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Router			/v1/endpoints/{endpoint_id} [delete]
func (ec *EndpointController) Delete(c *gin.Context) {
	endpointID := c.Param("endpoint_id")
	succMsg := "Endpoint delete triggered successfully"
	errMsg := "Failed to delete endpoint"
	userContext := getUserContext(c)
	forceDelete, parseErr := strconv.ParseBool(c.DefaultQuery("force", "false"))
	if parseErr != nil {
		err := &e.Error{Type: e.ParsingError, InternalErr: parseErr, Msg: errMsg}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: err})
		return
	}
	err := ec.endpointService.Delete(userContext, endpointID, forceDelete)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err})
}

// List godoc
//
//	@Summary		list
//	@Description	list endpoints
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			expand			query		[]string															false	"query param to denote what all extra fields to fetch"	collectionFormat(multi)
//	@Param			limit			query		int																	false	"limit"
//	@Param			offset			query		int																	false	"offset"
//	@Param			owner_id		query		string																false	"owner id for which the endpoints have to be returned"
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ListEndpoints}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/endpoints [get]
func (ec *EndpointController) List(c *gin.Context) {
	succMsg := "Endpoints fetched successfully"
	errMsg := "Failed to list endpoints"
	supportedQueryParams := []string{"owner_id"}
	listOptions, expandParams, err := GetQueryOptionsFromCtx(c, supportedQueryParams)
	if err != nil {
		err := &e.Error{Type: err.Type, InternalErr: err.InternalErr, Msg: fmt.Sprintf("%s: %s", errMsg, err.Msg), Log: err.Log}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: err})
		return
	}

	userContext := getUserContext(c)
	if err := ValidateOwner(userContext, listOptions, "endpoint"); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err})
		return
	}
	endpoints, totalCount, err := ec.endpointService.List(userContext, expandParams, listOptions)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err, Data: view.ListEndpointsResponse(endpoints, totalCount)})
}

// ListAPIKeys godoc
//
//	@Summary		listAPIKeys
//	@Description	list api keys by endpoint id
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			endpoint_id		path		string																true	"endpoint id"
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ListAPIKeys}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/endpoints/apikeys/{endpoint_id} [get]
func (ec *EndpointController) ListAPIKeys(c *gin.Context) {
	succMsg := "API keys for provided endpoint fetched successfully"
	endpointID := c.Param("endpoint_id")
	userContext := getUserContext(c)
	apiKeys, err := ec.endpointService.ListAPIKeysByEndpoint(userContext, endpointID)
	totalCount := int64(len(apiKeys))
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err, Data: view.ListAPIKeysResponse(apiKeys, totalCount)})
}

// // Update godoc
// //
// //	@Summary		update
// //	@Description	update an existing endpoint
// //	@Tags			endpoints
// //	@Accept			json
// //	@Produce		json
// //	@Param			endpoint_id		path		string								true	"endpoint id"
// //	@Param			Authorization	header		string								true	"access token sent via headers"
// //	@Param			endpoint		body		dto.UpdateEndpointRequest			true	"update existing endpoint object"
// //	@Success		200				{object}	response.HTTPSuccessResponseModel	"success response"
// //	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
// //	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
// //	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
// //	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
// //	@Router			/v1/endpoints/{endpoint_id} [patch]
// func (ec *EndpointController) Update(c *gin.Context) {
// 	var endpoint dto.UpdateEndpointRequest
// 	errMsg := "failed to update endpoint"
// 	succMsg := "endpoint update triggered successfully"
// 	if err := c.ShouldBindJSON(&endpoint); err != nil {
// 		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
// 		return
// 	}

// 	if err := ec.validator.Struct(endpoint); err != nil {
// 		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
// 		return
// 	}
// 	userContext := getUserContext(c)
// 	err := ec.endpointService.Update(userContext, c.Param("endpoint_id"), endpoint)
// 	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err})
// }

// ValidateEndpoint godoc
//
//	@Summary		validateEndpoint
//	@Description	validate endpoint object
//	@Tags			endpoints
//	@Accept			json
//	@Produce		json
//	@Param			endpoint_name	query		string								true	"endpoint name"
//	@Param			Authorization	header		string								true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessResponseModel	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Router			/v1/endpoints/validate [post]
func (ec *EndpointController) ValidateEndpoint(c *gin.Context) {
	endpointName := c.Query("endpoint_name")
	succMsg := "Endpoint name validated successfully"
	errMsg := "Invalid endpoint name"
	if len(endpointName) > constants.KserveISVCNameLength {
		fieldValidationErr := &e.FieldValidationErrorList{}
		logMsg := fmt.Sprintf("Length of name should be less than %d for the name %s", constants.KserveISVCNameLength, endpointName)
		fieldValidationErr.Errors = append(fieldValidationErr.Errors, e.FieldValidationError{Field: "name", ErrMsg: logMsg})
		err := &e.Error{Type: e.ValidationError, InternalErr: fieldValidationErr, Msg: errMsg, Log: logMsg}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err})
		return
	}
	err := ec.endpointService.ValidateEndpointName(endpointName)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: ec.logger, SuccMsg: succMsg, Err: err})
}
