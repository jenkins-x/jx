# \PullrequestsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugDefaultReviewersGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugDefaultReviewersGet) | **Get** /repositories/{username}/{repo_slug}/default-reviewers | 
[**RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameDelete**](PullrequestsApi.md#RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameDelete) | **Delete** /repositories/{username}/{repo_slug}/default-reviewers/{target_username} | 
[**RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameGet) | **Get** /repositories/{username}/{repo_slug}/default-reviewers/{target_username} | 
[**RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernamePut**](PullrequestsApi.md#RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernamePut) | **Put** /repositories/{username}/{repo_slug}/default-reviewers/{target_username} | 
[**RepositoriesUsernameRepoSlugPullrequestsActivityGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsActivityGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/activity | 
[**RepositoriesUsernameRepoSlugPullrequestsGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests | 
[**RepositoriesUsernameRepoSlugPullrequestsPost**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPost) | **Post** /repositories/{username}/{repo_slug}/pullrequests | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdActivityGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdActivityGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/activity | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApproveDelete**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApproveDelete) | **Delete** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/approve | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApprovePost**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApprovePost) | **Post** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/approve | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsCommentIdGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsCommentIdGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/comments/{comment_id} | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/comments | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/commits | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDeclinePost**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDeclinePost) | **Post** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/decline | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDiffGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDiffGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/diff | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id} | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost) | **Post** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/merge | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPatchGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPatchGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/patch | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPut**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPut) | **Put** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id} | 
[**RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet**](PullrequestsApi.md#RepositoriesUsernameRepoSlugPullrequestsPullRequestIdStatusesGet) | **Get** /repositories/{username}/{repo_slug}/pullrequests/{pull_request_id}/statuses | 


# **RepositoriesUsernameRepoSlugDefaultReviewersGet**
> RepositoriesUsernameRepoSlugDefaultReviewersGet(ctx, username, repoSlug)


Returns the repository's default reviewers.  These are the users that are automatically added as reviewers on every new pull request that is created.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameDelete**
> ModelError RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameDelete(ctx, username, targetUsername, repoSlug)


Removes a default reviewer from the repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **targetUsername** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameGet**
> ModelError RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernameGet(ctx, username, targetUsername, repoSlug)


Returns the specified reviewer.  This can be used to test whether a user is among the repository's default reviewers list. A 404 indicates that that specified user is not a default reviewer.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **targetUsername** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernamePut**
> ModelError RepositoriesUsernameRepoSlugDefaultReviewersTargetUsernamePut(ctx, username, targetUsername, repoSlug)


Adds the specified user to the repository's list of default reviewers.  This method is idempotent. Adding a user a second time has no effect.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **targetUsername** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsActivityGet**
> RepositoriesUsernameRepoSlugPullrequestsActivityGet(ctx, username, repoSlug, pullRequestId)


Returns a paginated list of the pull request's activity log.  This includes comments that were made by the reviewers, updates and approvals.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
  **pullRequestId** | **int32**| The id of the pull request.  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsGet**
> PaginatedPullrequests RepositoriesUsernameRepoSlugPullrequestsGet(ctx, username, repoSlug, optional)


Returns a paginated list of all pull requests on the specified repository. By default only open pull requests are returned. This can be controlled using the `state` query parameter. To retrieve pull requests that are in one of multiple states, repeat the `state` parameter for each individual state.  This endpoint also supports filtering and sorting of the results. See [filtering and sorting](../../../../meta/filtering) for more details.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
 **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
 **state** | **string**| Only return pull requests that are in this state. This parameter can be repeated. | 

### Return type

[**PaginatedPullrequests**](paginated_pullrequests.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPost**
> Pullrequest RepositoriesUsernameRepoSlugPullrequestsPost(ctx, username, repoSlug, optional)


Creates a new pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
 **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
 **body** | [**Pullrequest**](Pullrequest.md)| The new pull request.  The request URL you POST to becomes the destination repository URL. For this reason, you must specify an explicit source repository in the request object if you want to pull from a different repository (fork).  Since not all elements are required or even mutable, you only need to include the elements you want to initialize, such as the source branch and the title. | 

### Return type

[**Pullrequest**](pullrequest.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdActivityGet**
> RepositoriesUsernameRepoSlugPullrequestsPullRequestIdActivityGet(ctx, username, repoSlug, pullRequestId)


Returns a paginated list of the pull request's activity log.  This includes comments that were made by the reviewers, updates and approvals.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
  **pullRequestId** | **int32**| The id of the pull request.  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApproveDelete**
> RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApproveDelete(ctx, username, pullRequestId, repoSlug)


Redact the authenticated user's approval of the specified pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApprovePost**
> Participant RepositoriesUsernameRepoSlugPullrequestsPullRequestIdApprovePost(ctx, username, pullRequestId, repoSlug)


Approve the specified pull request as the authenticated user.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**Participant**](participant.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsCommentIdGet**
> PullrequestComment RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsCommentIdGet(ctx, username, pullRequestId, commentId, repoSlug)


Returns a specific pull request comment.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **commentId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**PullrequestComment**](pullrequest_comment.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsGet**
> PaginatedPullrequestComments RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommentsGet(ctx, username, repoSlug, pullRequestId)


Returns a paginated list of the pull request's comments.  This includes both global, inline comments and replies.  The default sorting is oldest to newest and can be overridden with the `sort` query parameter.  This endpoint also supports filtering and sorting of the results. See [filtering and sorting](../../../../../../meta/filtering) for more details.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the account, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
  **pullRequestId** | **int32**| The id of the pull request.  | 

### Return type

[**PaginatedPullrequestComments**](paginated_pullrequest_comments.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet**
> map[string]interface{} RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(ctx, username, pullRequestId, repoSlug)


Returns a paginated list of the pull request's commits.  These are the commits that are being merged into the destination branch when the pull requests gets accepted.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**map[string]interface{}**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDeclinePost**
> Pullrequest RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDeclinePost(ctx, username, pullRequestId, repoSlug)


Declines the pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**Pullrequest**](pullrequest.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDiffGet**
> ModelError RepositoriesUsernameRepoSlugPullrequestsPullRequestIdDiffGet(ctx, username, pullRequestId, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet**
> Pullrequest RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(ctx, username, repoSlug, pullRequestId)


Returns the specified pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the account, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
  **pullRequestId** | **int32**| The id of the pull request.  | 

### Return type

[**Pullrequest**](pullrequest.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost**
> Pullrequest RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost(ctx, username, pullRequestId, repoSlug, optional)


Merges the pull request.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**|  | 
 **pullRequestId** | **string**|  | 
 **repoSlug** | **string**|  | 
 **body** | [**PullrequestMergeParameters**](PullrequestMergeParameters.md)|  | 

### Return type

[**Pullrequest**](pullrequest.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPatchGet**
> ModelError RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPatchGet(ctx, username, pullRequestId, repoSlug)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **pullRequestId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPut**
> Pullrequest RepositoriesUsernameRepoSlugPullrequestsPullRequestIdPut(ctx, username, repoSlug, pullRequestId, optional)


Mutates the specified pull request.  This can be used to change the pull request's branches or description.  Only open pull requests can be mutated.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
  **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
  **pullRequestId** | **int32**| The id of the open pull request.  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| This can either be the username or the UUID of the user, surrounded by curly-braces, for example: &#x60;{user UUID}&#x60;.  | 
 **repoSlug** | **string**| This can either be the repository slug or the UUID of the repository, surrounded by curly-braces, for example: &#x60;{repository UUID}&#x60;.  | 
 **pullRequestId** | **int32**| The id of the open pull request.  | 
 **body** | [**Pullrequest**](Pullrequest.md)| The pull request that is to be updated. | 

### Return type

[**Pullrequest**](pullrequest.md)

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

