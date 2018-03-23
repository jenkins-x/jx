# \BranchrestrictionsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugBranchRestrictionsGet**](BranchrestrictionsApi.md#RepositoriesUsernameRepoSlugBranchRestrictionsGet) | **Get** /repositories/{username}/{repo_slug}/branch-restrictions | 
[**RepositoriesUsernameRepoSlugBranchRestrictionsIdDelete**](BranchrestrictionsApi.md#RepositoriesUsernameRepoSlugBranchRestrictionsIdDelete) | **Delete** /repositories/{username}/{repo_slug}/branch-restrictions/{id} | 
[**RepositoriesUsernameRepoSlugBranchRestrictionsIdGet**](BranchrestrictionsApi.md#RepositoriesUsernameRepoSlugBranchRestrictionsIdGet) | **Get** /repositories/{username}/{repo_slug}/branch-restrictions/{id} | 
[**RepositoriesUsernameRepoSlugBranchRestrictionsIdPut**](BranchrestrictionsApi.md#RepositoriesUsernameRepoSlugBranchRestrictionsIdPut) | **Put** /repositories/{username}/{repo_slug}/branch-restrictions/{id} | 
[**RepositoriesUsernameRepoSlugBranchRestrictionsPost**](BranchrestrictionsApi.md#RepositoriesUsernameRepoSlugBranchRestrictionsPost) | **Post** /repositories/{username}/{repo_slug}/branch-restrictions | 


# **RepositoriesUsernameRepoSlugBranchRestrictionsGet**
> PaginatedBranchrestrictions RepositoriesUsernameRepoSlugBranchRestrictionsGet(ctx, username, repoSlug)


Returns a paginated list of all branch restrictions on the repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**PaginatedBranchrestrictions**](paginated_branchrestrictions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugBranchRestrictionsIdDelete**
> RepositoriesUsernameRepoSlugBranchRestrictionsIdDelete(ctx, username, repoSlug, id)


Deletes an existing branch restriction rule.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **id** | **string**| The restriction rule&#39;s id | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugBranchRestrictionsIdGet**
> Branchrestriction RepositoriesUsernameRepoSlugBranchRestrictionsIdGet(ctx, username, repoSlug, id)


Returns a specific branch restriction rule.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **id** | **string**| The restriction rule&#39;s id | 

### Return type

[**Branchrestriction**](branchrestriction.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugBranchRestrictionsIdPut**
> Branchrestriction RepositoriesUsernameRepoSlugBranchRestrictionsIdPut(ctx, username, repoSlug, id, body)


Updates an existing branch restriction rule.  Fields not present in the request body are ignored.  See [`POST`](../../branch-restrictions#post) for details.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **id** | **string**| The restriction rule&#39;s id | 
  **body** | [**Branchrestriction**](Branchrestriction.md)| The new version of the existing rule | 

### Return type

[**Branchrestriction**](branchrestriction.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugBranchRestrictionsPost**
> Branchrestriction RepositoriesUsernameRepoSlugBranchRestrictionsPost(ctx, username, repoSlug, body)


Creates a new branch restriction rule for a repository.  `kind` describes what will be restricted. Allowed values are: `push`, `force`, `delete`, and `restrict_merges`.  Different kinds of branch restrictions have different requirements:  * `push` and `restrict_merges` require `users` and `groups` to be   specified. Empty lists are allowed, in which case permission is   denied for everybody. * `force` can not be specified in a Mercurial repository.  `pattern` is used to determine which branches will be restricted.  A `'*'` in `pattern` will expand to match zero or more characters, and every other character matches itself. For example, `'foo*'` will match `'foo'` and `'foobar'`, but not `'barfoo'`. `'*'` will match all branches.  `users` and `groups` are lists of user names and group names.  `kind` and `pattern` must be unique within a repository; adding new users or groups to an existing restriction should be done via `PUT`.  Note that branch restrictions with overlapping patterns are allowed, but the resulting behavior may be surprising.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **body** | [**Branchrestriction**](Branchrestriction.md)| The new rule | 

### Return type

[**Branchrestriction**](branchrestriction.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

