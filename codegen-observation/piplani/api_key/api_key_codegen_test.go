package main

import (
	"testing"
)

func TestCreate(t *testing.T) {
	akc := &APIKeyController{}
	c := &gin.Context{}
	errMsg := "failed to create new API Key"
	succMsg := "API Key created successfully"
	var apiKeyCreateRequest dto.APIKeyCreateRequest
	if err := c.ShouldBindJSON(&apiKeyCreateRequest); err != nil {
		require.Equal(t, errMsg, err.Error())
		return
	}
	if err := akc.validator.Struct(apiKeyCreateRequest); err != nil {
		require.Equal(t, errMsg, err.Error())
		return
	}
	userContext := getUserContext(c)
	resp, err := akc.APIKeyService.Create(userContext, apiKeyCreateRequest)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestList(t *testing.T) {
	akc := &APIKeyController{}
	c := &gin.Context{}
	succMsg := "API Key retrieved successfully"
	errMsg := "Failed to list API Keys"
	listOptions, _, err := GetQueryOptionsFromCtx(c, []string{"owner_id"})
	require.NoError(t, err)
	userContext := getUserContext(c)
	APIKeys, totalCount, err := akc.APIKeyService.List(userContext, listOptions)
	require.NoError(t, err)
	require.NotNil(t, APIKeys)
	require.NotNil(t, totalCount)
}

func TestDelete(t *testing.T) {
	akc := &APIKeyController{}
	c := &gin.Context{}
	apiKeyID := "apikey_id"
	succMsg := "API Key deleted successfully"
	err := akc.APIKeyService.Delete(getUserContext(c), apiKeyID)
	require.NoError(t, err)
}

func TestUpdate(t *testing.T) {
	akc := &APIKeyController{}
	c := &gin.Context{}
	apiKeyID := "apikey_id"
	succMsg := "API Key updated successfully"
	errMsg := "Failed to update API Key"
	var apiKeyUpdateRequest dto.APIKeyUpdateRequest
	if err := c.ShouldBindJSON(&apiKeyUpdateRequest); err != nil {
		require.Equal(t, errMsg, err.Error())
		return
	}
	if err := akc.validator.Struct(apiKeyUpdateRequest); err != nil {
		require.Equal(t, errMsg, err.Error())
		return
	}
	userContext := getUserContext(c)
	err := akc.APIKeyService.Update(userContext, apiKeyID, apiKeyUpdateRequest)
	require.NoError(t, err)
}

func TestValidateUniqueConstraints(t *testing.T) {
	akc := &APIKeyController{}
	userContext := &dto.UserContext{}
	apiKeyCreateRequest := &dto.APIKeyCreateRequest{}
	err := akc.validateUniqueConstraints(userContext, apiKeyCreateRequest, "failed to create new API Key")
	require.Nil(t, err)
}