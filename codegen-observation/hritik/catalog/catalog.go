package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	e "github.com/nutanix-core/nai-api/common/errors"
	"github.com/nutanix-core/nai-api/common/logger"
	"github.com/nutanix-core/nai-api/common/response"
	"github.com/nutanix-core/nai-api/iep/constants"
	dto "github.com/nutanix-core/nai-api/iep/internal/dto"
	auth "github.com/nutanix-core/nai-api/iep/internal/middleware"
	"github.com/nutanix-core/nai-api/iep/internal/service"
	"github.com/nutanix-core/nai-api/iep/internal/view"
)

// CatalogController represents a catalog controller
type CatalogController struct {
	v1Route        *gin.RouterGroup
	logger         logger.Logger
	validator      *validator.Validate
	catalogService service.ICatalogService
	authMiddleware auth.IAuthenticationMiddleware
}

// NewCatalogController function and initiates the route
func NewCatalogController(v1Route *gin.RouterGroup, logger logger.Logger, validator *validator.Validate, catalogService service.ICatalogService, authMiddleware auth.IAuthenticationMiddleware) *CatalogController {
	controller := &CatalogController{v1Route: v1Route, logger: logger, validator: validator, catalogService: catalogService, authMiddleware: authMiddleware}
	controller.route()
	return controller
}

// Catalogs will route a request to the correct function
func (cc *CatalogController) route() {
	route := cc.v1Route.Group("/catalogs")
	route.GET("/:catalog_id", cc.authMiddleware.ValidateAccessToken(auth.AllowAll), cc.GetByID)
	route.POST("", cc.authMiddleware.ValidateAccessToken(auth.AllowSuperAdmin), cc.Create)
	route.GET("", cc.authMiddleware.ValidateAccessToken(auth.AllowAll), cc.List)
	// route.PATCH("/:catalog_id", cc.authMiddleware.ValidateAccessToken(auth.AllowSuperAdmin), cc.Update)
	route.DELETE("/:catalog_id", cc.authMiddleware.ValidateAccessToken(auth.AllowSuperAdmin), cc.Delete)
	route.POST("/requirements", cc.authMiddleware.ValidateAccessToken(auth.AllowAll), cc.GetRequirements)
}

// Create godoc
//
//	@Summary		create
//	@Description	create a new catalog
//	@Tags			catalogs
//	@Accept			json
//	@Produce		json
//	@Param			catalog			body		dto.CreateCatalogRequest								true	"new create catalog request object"
//	@Param			Authorization	header		string													true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ID}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel						"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel						"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel						"forbidden response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel						"internal server error response"
//	@Router			/v1/catalogs [post]
func (cc *CatalogController) Create(c *gin.Context) {
	var catalog dto.CreateCatalogRequest
	errMsg := "Failed to create new catalog entry"
	if err := c.ShouldBindJSON(&catalog); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}
	if err := cc.validator.Struct(catalog); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}

	if err := cc.validateUniqueConstraints(catalog, errMsg); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: err})
		return
	}

	succMsg := "Catalog created successfully"
	id, appErr := cc.catalogService.Create(catalog)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: appErr, Data: view.GetID(id)})
}

// List godoc
//
//	@Summary		list
//	@Description	get catalogs by limit and offset
//	@Tags			catalogs
//	@Accept			json
//	@Produce		json
//	@Param			model_name		query		string																false	"model name"
//	@Param			deprecated		query		bool																false	"filter catalog items on deprecated field"
//	@Param			source_hub		query		enum.CatalogSourceHub												false	"filter catalog items on source hub"
//	@Param			limit			query		int																	false	"limit"
//	@Param			offset			query		int																	false	"offset"
//	@Param			Authorization	header		string																true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.ListCatalogs}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel									"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel									"unauthorized response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel									"internal server error response"
//	@Router			/v1/catalogs [get]
func (cc *CatalogController) List(c *gin.Context) {
	errMsg := "Failed to get catalogs"
	succMsg := "Catalogs fetched successfully"
	supportedQueryParams := []string{"model_name", "deprecated", "source_hub"}
	listOptions, _, err := GetQueryOptionsFromCtx(c, supportedQueryParams)
	if err != nil {
		err := &e.Error{Type: err.Type, InternalErr: err.InternalErr, Msg: fmt.Sprintf("%s: %s", errMsg, err.Msg), Log: err.Log}
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: err})
		return
	}

	setDeprecatedFalseIfUnset(&listOptions)

	catalogs, totalCount, err := cc.catalogService.List(listOptions)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: err, Data: view.ListCatalogsResponse(catalogs, totalCount)})
}

// GetByID godoc
//
//	@Summary		getByID
//	@Description	get a catalog by id
//	@Tags			catalogs
//	@Accept			json
//	@Produce		json
//	@Param			catalog_id		path		string															true	"catalog id"
//	@Param			Authorization	header		string															true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=view.Catalog}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel								"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel								"unauthorized response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel								"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel								"internal server error response"
//	@Router			/v1/catalogs/{catalog_id} [get]
func (cc *CatalogController) GetByID(c *gin.Context) {
	catalogID := c.Param("catalog_id")
	succMsg := "Catalog fetched successfully"
	catalog, appErr := cc.catalogService.GetByID(catalogID)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: appErr, Data: view.GetCatalogByIDResponse(catalog)})
}

// Delete godoc
//
//	@Summary		delete
//	@Description	delete catalog by id
//	@Tags			catalogs
//	@Accept			json
//	@Produce		json
//	@Param			catalog_id		path		string								true	"catalog id"
//	@Param			Authorization	header		string								true	"access token sent via headers"
//	@Success		200				{object}	response.HTTPSuccessResponseModel	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel	"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel	"unauthorized response"
//	@Failure		403				{object}	response.HTTPFailureResponseModel	"forbidden response"
//	@Failure		404				{object}	response.HTTPFailureResponseModel	"not found response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel	"internal server error response"
//	@Router			/v1/catalogs/{catalog_id} [delete]
func (cc *CatalogController) Delete(c *gin.Context) {
	catalogID := c.Param("catalog_id")
	succMsg := "Catalog deleted successfully"
	appErr := cc.catalogService.Delete(catalogID)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: appErr})
}

// Update endpoint is disabled
func (cc *CatalogController) Update(c *gin.Context) {
	var catalog dto.UpdateCatalogRequest
	catalogID := c.Param("catalog_id")

	errMsg := "Failed to update the catalog"
	succMsg := "Catalog updated successfully"

	if err := c.ShouldBindJSON(&catalog); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}

	if err := cc.validator.Struct(catalog); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}

	appErr := cc.catalogService.Update(catalogID, catalog)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: appErr})
}

// GetRequirements godoc
//
//	@Summary		requirements
//	@Description	get requirements of a model
//	@Tags			catalogs
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string																			true	"access token sent via headers"
//	@Param			catalog			body		dto.CatalogRequirements															true	"new catalog requirements object"
//	@Success		200				{object}	response.HTTPSuccessWithDataResponseModel{data=dto.CatalogRequirementsResponse}	"success response"
//	@Failure		400				{object}	response.HTTPFailureResponseModel												"bad request response"
//	@Failure		401				{object}	response.HTTPFailureResponseModel												"unauthorized response"
//	@Failure		500				{object}	response.HTTPFailureResponseModel												"internal server error response"
//	@Router			/v1/catalogs/requirements [post]
func (cc *CatalogController) GetRequirements(c *gin.Context) {
	errMsg := "Failed to get catalog requirements"
	var catalogRequirements dto.CatalogRequirements
	if err := c.ShouldBindJSON(&catalogRequirements); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.BindingError, InternalErr: err, Msg: errMsg}})
		return
	}
	catalogRequirements.SetDefaults()
	if err := cc.validator.Struct(catalogRequirements); err != nil {
		response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, Err: &e.Error{Type: e.ValidationError, InternalErr: err, Msg: errMsg}})
		return
	}
	succMsg := "Catalog requirements fetched successfully"
	requirementResponse, err := cc.catalogService.GetRequirements(catalogRequirements)
	response.HTTPResponse(response.HTTPResponseOptions{Ctx: c, Logger: cc.logger, SuccMsg: succMsg, Err: err, Data: requirementResponse})
}

func (cc *CatalogController) validateUniqueConstraints(catalog dto.CreateCatalogRequest, errMsg string) (err *e.Error) {
	supportedQueryParams := []string{constants.ModelName, constants.ModelRevision, constants.CreatedBy}
	var listOptions dto.ListOptions
	listOptions.AddEqualToFiltersFromMap(map[string][]string{
		constants.ModelName:     {catalog.ModelName},
		constants.ModelRevision: {catalog.ModelRevision},
		constants.CreatedBy:     {constants.SuperAdmin},
	}, supportedQueryParams)
	listResponse, _, err := cc.catalogService.List(listOptions)
	if err != nil {
		return err
	} else if len(listResponse) > 0 {
		modelNameErr := e.FieldValidationError{
			Field:  "modelName",
			ErrMsg: "ModelName with same ModelRevision already exists, provide different ModelName or ModelRevision",
		}

		internalValidationErr := &e.FieldValidationErrorList{
			Errors: []e.FieldValidationError{modelNameErr},
		}
		return &e.Error{Type: e.ValidationError, InternalErr: internalValidationErr, Msg: errMsg}
	}

	return nil
}

func setDeprecatedFalseIfUnset(listOptions *dto.ListOptions) {
	if listOptions.GetFilterFromField("deprecated").Field != "" {
		return
	}
	listOptions.AddEqualToFiltersFromMap(map[string][]string{
		"deprecated": {"false"},
	}, []string{"deprecated"})
}
