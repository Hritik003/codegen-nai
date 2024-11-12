package v1

import (
	"fmt"

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

// APIKeyController represents a APIKey controller
type APIKeyController struct {
	v1Route        *gin.RouterGroup
	logger         logger.Logger
	validator      *validator.Validate
	APIKeyService  service.IAPIKeyService
	authMiddleware auth.IAuthenticationMiddleware
}

// NewAPIKeyController function and initiates the route
func NewAPIKeyController(v1Route *gin.RouterGroup, logger logger.Logger, validator *validator.Validate, apiKeyService service.IAPIKeyService, authMiddleware auth.IAuthenticationMiddleware) *APIKeyController {
	controller := &APIKeyController{v1Route: v1Route, logger: logger, validator: validator, APIKeyService: apiKeyService, authMiddleware: authMiddleware}
	controller.route()
	return controller
}

// Users will route a request to the correct function
func (akc *APIKeyController) route() {
	route := akc.v1Route.Group("/apikeys", akc.authMiddleware.ValidateAccessToken(auth.AllowAll))
	route.POST("", akc.Create)
	route.GET("", akc.List)
	route.DELETE("/:apikey_id", akc.Delete)
	route.PATCH("/:apikey_id", akc.Update)
}

// Create godoc
//
//	@Summary		create
//	@Description	create a new apikey
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			apiKeyCreateRequest	body		dto.APIKeyCreateRequest												true	"new apikey create request object"
//	@Param			Authorization		header		string																true	"access token sent via headers"
//	@Success		200					{object}	response.HTTPSuccessWithDataResponseModel{data=view.APIKeyValue}	"success response"
//	@Failure		400					{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401					{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403					{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		500					{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/apikeys [post]
func (akc *APIKeyController) Create(c *gin.Context) {
	errMsg := "failed to create new API Key"
	succMsg := "API Key created successfully"
	userContext := getUserContext(c)

	var apiKeyCreateRequest dto.APIKeyCreateRequest
	if err := c.ShouldBindJSON(&apiKeyCreateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}

	if err := akc.validator.Struct(apiKeyCreateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}

	if err := akc.validateUniqueConstraints(userContext, apiKeyCreateRequest, errMsg); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: err})
		return
	}

	resp, err := akc.APIKeyService.Create(userContext, apiKeyCreateRequest)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, SuccMsg: succMsg, Err: err, Data: view.GetAPIKeyValue(resp.GeneratedKey, resp.ID)})
}

// List godoc
//
//	@Summary		list
//	@Description	list apikeys
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			limit			query		int																	false	"limit"
//	@Param			offset			query		int																	false	"offset"
//	@Param			owner_id		query		string																false	"owner id for which the apikeys have to be returned"
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ListAPIKeys}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/apikeys [get]
func (akc *APIKeyController) List(c *gin.Context) {
	succMsg := "API Key retrieved successfully"
	errMsg := "Failed to list API Keys"
	supportedQueryParams := []string{"owner_id"}
	listOptions, _, err := GetQueryOptionsFromCtx(c, supportedQueryParams)
	if err != nil {
		err := &e.Error{Type: err.Type, InternalErr: err.InternalErr, Msg: fmt.Sprintf("%s: %s", errMsg, err.Msg), Log: err.Log}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: err})
		return
	}

	userContext := getUserContext(c)
	if err := ValidateOwner(userContext, listOptions, "api keys"); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, SuccMsg: succMsg, Err: err})
		return
	}
	APIKeys, totalCount, err := akc.APIKeyService.List(userContext, listOptions)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, SuccMsg: succMsg, Err: err, Data: view.ListAPIKeysResponse(APIKeys, totalCount)})
}

// Delete godoc
//
//	@Summary		delete
//	@Description	delete apikey with id
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			apikey_id		path		string								true	"apikey id"
//	@Param			Authorization	header		string								true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessResponseModel	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel	"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Router			/v1/apikeys/{apikey_id} [delete]
func (akc *APIKeyController) Delete(c *gin.Context) {
	apiKeyID := c.Param("apikey_id")
	userContext := getUserContext(c)
	succMsg := "API Key deleted successfully"
	err := akc.APIKeyService.Delete(userContext, apiKeyID)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, SuccMsg: succMsg, Err: err})
}

// Update godoc
//
//	@Summary		update
//	@Description	update apikey with status and endpoint id list
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			apikey_id			path		string																true	"apikey id"
//	@Param			apiKeyUpdateRequest	body		dto.APIKeyUpdateRequest												true	"apikey update request object"
//	@Param			Authorization		header		string																true	"access token sent via headers"
//	@Success		200					{object}	response.HTTPSuccessWithDataResponseModel{data=view.APIKeyValue}	"success response"
//	@Failure		400					{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401					{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		403					{object}	response.HTTPFailureResponseModel									"forbidden response"
//	@Failure		404					{object}	response.HTTPFailureResponseModel									"not found response"
//	@Failure		500					{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/apikeys/{apikey_id} [patch]
func (akc *APIKeyController) Update(c *gin.Context) {
	errMsg := "Failed to update API Key"
	succMsg := "API Key updated successfully"
	apiKeyID := c.Param("apikey_id")

	var apiKeyUpdateRequest dto.APIKeyUpdateRequest
	if err := c.ShouldBindJSON(&apiKeyUpdateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}

	if err := akc.validator.Struct(apiKeyUpdateRequest); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}

	userContext := getUserContext(c)
	err := akc.APIKeyService.Update(userContext, apiKeyID, apiKeyUpdateRequest)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: akc.logger, SuccMsg: succMsg, Err: err})
}

func (akc *APIKeyController) validateUniqueConstraints(userContext dto.UserContext, apiKeyCreateRequest dto.APIKeyCreateRequest, errMsg string) (err *e.Error) {
	var listOptions dto.ListOptions
	supportedFields := []string{constants.Name, constants.OwnerID}
	listOptions.AddEqualToFiltersFromMap(map[string][]string{
		constants.Name:    {apiKeyCreateRequest.Name},
		constants.OwnerID: {userContext.UserID},
	}, supportedFields)
	listResponse, _, err := akc.APIKeyService.List(dto.UserContext{}, listOptions)
	if err != nil {
		return err
	} else if len(listResponse) > 0 {
		field := "name"
		fieldErrMsg := fmt.Sprintf("%s already exists, please provide a different %s", field, field)
		nameErr := e.FieldValidationError{
			Field:  field,
			ErrMsg: fieldErrMsg,
		}
		internalValidationErr := &e.FieldValidationErrorList{
			Errors: []e.FieldValidationError{nameErr},
		}
		return &e.Error{Type: e.ValidationError, InternalErr: internalValidationErr, Msg: errMsg}
	}

	return nil
}
