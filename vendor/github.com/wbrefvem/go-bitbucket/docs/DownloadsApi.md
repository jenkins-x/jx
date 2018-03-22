# \DownloadsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugDownloadsFilenameDelete**](DownloadsApi.md#RepositoriesUsernameRepoSlugDownloadsFilenameDelete) | **Delete** /repositories/{username}/{repo_slug}/downloads/{filename} | 
[**RepositoriesUsernameRepoSlugDownloadsFilenameGet**](DownloadsApi.md#RepositoriesUsernameRepoSlugDownloadsFilenameGet) | **Get** /repositories/{username}/{repo_slug}/downloads/{filename} | 
[**RepositoriesUsernameRepoSlugDownloadsGet**](DownloadsApi.md#RepositoriesUsernameRepoSlugDownloadsGet) | **Get** /repositories/{username}/{repo_slug}/downloads | 
[**RepositoriesUsernameRepoSlugDownloadsPost**](DownloadsApi.md#RepositoriesUsernameRepoSlugDownloadsPost) | **Post** /repositories/{username}/{repo_slug}/downloads | 


# **RepositoriesUsernameRepoSlugDownloadsFilenameDelete**
> ModelError RepositoriesUsernameRepoSlugDownloadsFilenameDelete(ctx, username, filename, repoSlug)


Deletes the specified download artifact from the repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **filename** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDownloadsFilenameGet**
> ModelError RepositoriesUsernameRepoSlugDownloadsFilenameGet(ctx, username, filename, repoSlug)


Return a redirect to the contents of a download artifact.  This endpoint returns the actual file contents and not the artifact's metadata.      $ curl -s -L https://api.bitbucket.org/2.0/repositories/evzijst/git-tests/downloads/hello.txt     Hello World

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **filename** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDownloadsGet**
> ModelError RepositoriesUsernameRepoSlugDownloadsGet(ctx, username, repoSlug)


Returns a list of download links associated with the repository.

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

# **RepositoriesUsernameRepoSlugDownloadsPost**
> ModelError RepositoriesUsernameRepoSlugDownloadsPost(ctx, username, repoSlug)


Upload new download artifacts.  To upload files, perform a `multipart/form-data` POST containing one or more `files` fields:      $ echo Hello World > hello.txt     $ curl -s -u evzijst -X POST https://api.bitbucket.org/2.0/repositories/evzijst/git-tests/downloads -F files=@hello.txt  When a file is uploaded with the same name as an existing artifact, then the existing file will be replaced.

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

