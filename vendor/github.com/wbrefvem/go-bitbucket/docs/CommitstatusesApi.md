# \CommitstatusesApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyGet**](CommitstatusesApi.md#RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyGet) | **Get** /repositories/{username}/{repo_slug}/commit/{node}/statuses/build/{key} | 
[**RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyPut**](CommitstatusesApi.md#RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyPut) | **Put** /repositories/{username}/{repo_slug}/commit/{node}/statuses/build/{key} | 
[**RepositoriesUsernameRepoSlugCommitNodeStatusesBuildPost**](CommitstatusesApi.md#RepositoriesUsernameRepoSlugCommitNodeStatusesBuildPost) | **Post** /repositories/{username}/{repo_slug}/commit/{node}/statuses/build | 
[**RepositoriesUsernameRepoSlugCommitNodeStatusesGet**](CommitstatusesApi.md#RepositoriesUsernameRepoSlugCommitNodeStatusesGet) | **Get** /repositories/{username}/{repo_slug}/commit/{node}/statuses | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet**](CommitstatusesApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/statuses | 


# **RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyGet**
> Commitstatus RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyGet(ctx, username, repoSlug, node, key)


Returns the specified build status for a commit.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1 | 
  **key** | **string**| The build status&#39; unique key | 

### Return type

[**Commitstatus**](commitstatus.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyPut**
> Commitstatus RepositoriesUsernameRepoSlugCommitNodeStatusesBuildKeyPut(ctx, username, repoSlug, node, key, optional)


Used to update the current status of a build status object on the specific commit.  This operation can also be used to change other properties of the build status:  * `state` * `name` * `description` * `url` * `refname`  The `key` cannot be changed.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1 | 
  **key** | **string**| The commit status&#39; unique key | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**|  | 
 **repoSlug** | **string**|  | 
 **node** | **string**| The commit&#39;s SHA1 | 
 **key** | **string**| The commit status&#39; unique key | 
 **body** | [**Commitstatus**](Commitstatus.md)| The updated build status object | 

### Return type

[**Commitstatus**](commitstatus.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitNodeStatusesBuildPost**
> Commitstatus RepositoriesUsernameRepoSlugCommitNodeStatusesBuildPost(ctx, username, repoSlug, node, optional)


Creates a new build status against the specified commit.  If the specified key already exists, the existing status object will be overwritten.  When creating a new commit status, you can use a URI template for the URL. Templates are URLs that contain variable names that Bitbucket will evaluate at runtime whenever the URL is displayed anywhere similar to parameter substitution in [Bitbucket Connect](https://developer.atlassian.com/bitbucket/concepts/context-parameters.html). For example, one could use `https://foo.com/builds/{repository.full_name}` which Bitbucket will turn into `https://foo.com/builds/foo/bar` at render time. The context variables available are `repository` and `commit`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1 | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**|  | 
 **repoSlug** | **string**|  | 
 **node** | **string**| The commit&#39;s SHA1 | 
 **body** | [**Commitstatus**](Commitstatus.md)| The new commit status object. | 

### Return type

[**Commitstatus**](commitstatus.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitNodeStatusesGet**
> PaginatedCommitstatuses RepositoriesUsernameRepoSlugCommitNodeStatusesGet(ctx, username, repoSlug, node)


Returns all statuses (e.g. build results) for a specific commit.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1 | 

### Return type

[**PaginatedCommitstatuses**](paginated_commitstatuses.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet**
> PaginatedCommitstatuses RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet(ctx, username, repoSlug, pullRequestId)


Returns all statuses (e.g. build results) for the given pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **pullRequestId** | **int32**| The pull request&#39;s id | 

### Return type

[**PaginatedCommitstatuses**](paginated_commitstatuses.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

