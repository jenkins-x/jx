# \RefsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugRefsBranchesGet**](RefsApi.md#RepositoriesUsernameRepoSlugRefsBranchesGet) | **Get** /repositories/{username}/{repo_slug}/refs/branches | 
[**RepositoriesUsernameRepoSlugRefsBranchesNameGet**](RefsApi.md#RepositoriesUsernameRepoSlugRefsBranchesNameGet) | **Get** /repositories/{username}/{repo_slug}/refs/branches/{name} | 
[**RepositoriesUsernameRepoSlugRefsGet**](RefsApi.md#RepositoriesUsernameRepoSlugRefsGet) | **Get** /repositories/{username}/{repo_slug}/refs | 
[**RepositoriesUsernameRepoSlugRefsTagsGet**](RefsApi.md#RepositoriesUsernameRepoSlugRefsTagsGet) | **Get** /repositories/{username}/{repo_slug}/refs/tags | 
[**RepositoriesUsernameRepoSlugRefsTagsNameGet**](RefsApi.md#RepositoriesUsernameRepoSlugRefsTagsNameGet) | **Get** /repositories/{username}/{repo_slug}/refs/tags/{name} | 
[**RepositoriesUsernameRepoSlugRefsTagsPost**](RefsApi.md#RepositoriesUsernameRepoSlugRefsTagsPost) | **Post** /repositories/{username}/{repo_slug}/refs/tags | 


# **RepositoriesUsernameRepoSlugRefsBranchesGet**
> ModelError RepositoriesUsernameRepoSlugRefsBranchesGet(ctx, username, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugRefsBranchesNameGet**
> ModelError RepositoriesUsernameRepoSlugRefsBranchesNameGet(ctx, username, name, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **name** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugRefsGet**
> ModelError RepositoriesUsernameRepoSlugRefsGet(ctx, username, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugRefsTagsGet**
> ModelError RepositoriesUsernameRepoSlugRefsTagsGet(ctx, username, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  The username for the owner of the repository. This can either be the &#x60;username&#x60; of the owner or the &#x60;UUID&#x60; of the owner (surrounded by curly-braces (&#x60;{}&#x60;)). Owners can be users or teams.  | 
  **repoSlug** | **string**|  The repo slug for the repository.  This can either be the &#x60;repo_slug&#x60; of the repository or the &#x60;UUID&#x60; of the repository (surrounded by curly-braces (&#x60;{}&#x60;))  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugRefsTagsNameGet**
> ModelError RepositoriesUsernameRepoSlugRefsTagsNameGet(ctx, username, name, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **name** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugRefsTagsPost**
> Tag RepositoriesUsernameRepoSlugRefsTagsPost(ctx, username, repoSlug, body)


Creates a new tag in the specified repository.  The payload of the POST should consist of a JSON document that contains the name of the tag and the target hash.  ``` {     \"name\" : \"new tag name\",     \"target\" : {         \"hash\" : \"target commit hash\",     } } ```  This endpoint does support using short hash prefixes for the commit hash, but it may return a 400 response if the provided prefix is ambiguous. Using a full commit hash is the preferred approach.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  The username for the owner of the repository. This can either be the &#x60;username&#x60; of the owner or the &#x60;UUID&#x60; of the owner (surrounded by curly-braces (&#x60;{}&#x60;)). Owners can be users or teams.  | 
  **repoSlug** | **string**|  The repo slug for the repository.  This can either be the &#x60;repo_slug&#x60; of the repository or the &#x60;UUID&#x60; of the repository (surrounded by curly-braces (&#x60;{}&#x60;))  | 
  **body** | [**Tag**](Tag.md)|  | 

### Return type

[**Tag**](tag.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

