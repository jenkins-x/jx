# \CommitsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugCommitNodeApproveDelete**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitNodeApproveDelete) | **Delete** /repositories/{username}/{repo_slug}/commit/{node}/approve | 
[**RepositoriesUsernameRepoSlugCommitNodeApprovePost**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitNodeApprovePost) | **Post** /repositories/{username}/{repo_slug}/commit/{node}/approve | 
[**RepositoriesUsernameRepoSlugCommitRevisionGet**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitRevisionGet) | **Get** /repositories/{username}/{repo_slug}/commit/{revision} | 
[**RepositoriesUsernameRepoSlugCommitShaCommentsCommentIdGet**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitShaCommentsCommentIdGet) | **Get** /repositories/{username}/{repo_slug}/commit/{sha}/comments/{comment_id} | 
[**RepositoriesUsernameRepoSlugCommitShaCommentsGet**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitShaCommentsGet) | **Get** /repositories/{username}/{repo_slug}/commit/{sha}/comments | 
[**RepositoriesUsernameRepoSlugCommitsGet**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitsGet) | **Get** /repositories/{username}/{repo_slug}/commits | 
[**RepositoriesUsernameRepoSlugCommitsPost**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitsPost) | **Post** /repositories/{username}/{repo_slug}/commits | 
[**RepositoriesUsernameRepoSlugCommitsRevisionGet**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitsRevisionGet) | **Get** /repositories/{username}/{repo_slug}/commits/{revision} | 
[**RepositoriesUsernameRepoSlugCommitsRevisionPost**](CommitsApi.md#RepositoriesUsernameRepoSlugCommitsRevisionPost) | **Post** /repositories/{username}/{repo_slug}/commits/{revision} | 
[**RepositoriesUsernameRepoSlugDiffSpecGet**](CommitsApi.md#RepositoriesUsernameRepoSlugDiffSpecGet) | **Get** /repositories/{username}/{repo_slug}/diff/{spec} | 
[**RepositoriesUsernameRepoSlugPatchSpecGet**](CommitsApi.md#RepositoriesUsernameRepoSlugPatchSpecGet) | **Get** /repositories/{username}/{repo_slug}/patch/{spec} | 


# **RepositoriesUsernameRepoSlugCommitNodeApproveDelete**
> RepositoriesUsernameRepoSlugCommitNodeApproveDelete(ctx, username, repoSlug, node)


Redact the authenticated user's approval of the specified commit.  This operation is only available to users that have explicit access to the repository. In contrast, just the fact that a repository is publicly accessible to users does not give them the ability to approve commits.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1. | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitNodeApprovePost**
> Participant RepositoriesUsernameRepoSlugCommitNodeApprovePost(ctx, username, repoSlug, node)


Approve the specified commit as the authenticated user.  This operation is only available to users that have explicit access to the repository. In contrast, just the fact that a repository is publicly accessible to users does not give them the ability to approve commits.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **node** | **string**| The commit&#39;s SHA1. | 

### Return type

[**Participant**](participant.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitRevisionGet**
> Commit RepositoriesUsernameRepoSlugCommitRevisionGet(ctx, username, repoSlug, revision)


Returns the specified commit.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **revision** | **string**| The commit&#39;s SHA1. | 

### Return type

[**Commit**](commit.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitShaCommentsCommentIdGet**
> ModelError RepositoriesUsernameRepoSlugCommitShaCommentsCommentIdGet(ctx, username, sha, commentId, repoSlug)


Returns the specified commit comment.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **sha** | **string**|  | 
  **commentId** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitShaCommentsGet**
> ModelError RepositoriesUsernameRepoSlugCommitShaCommentsGet(ctx, username, sha, repoSlug)


Returns the commit's comments.  This includes both global and inline comments.  The default sorting is oldest to newest and can be overridden with the `sort` query parameter.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **sha** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitsGet**
> ModelError RepositoriesUsernameRepoSlugCommitsGet(ctx, username, repoSlug)


These are the repository's commits. They are paginated and returned in reverse chronological order, similar to the output of `git log` and `hg log`. Like these tools, the DAG can be filtered.  ## GET /repositories/{username}/{repo_slug}/commits/  Returns all commits in the repo in topological order (newest commit first). All branches and tags are included (similar to `git log --all` and `hg log`).  ## GET /repositories/{username}/{repo_slug}/commits/master  Returns all commits on rev `master` (similar to `git log master`, `hg log master`).  ## GET /repositories/{username}/{repo_slug}/commits/dev?exclude=master  Returns all commits on ref `dev`, except those that are reachable on `master` (similar to `git log dev ^master`).  ## GET /repositories/{username}/{repo_slug}/commits/?exclude=master  Returns all commits in the repo that are not on master (similar to `git log --all ^master`).  ## GET /repositories/{username}/{repo_slug}/commits/?include=foo&include=bar&exclude=fu&exclude=fubar  Returns all commits that are on refs `foo` or `bar`, but not on `fu` or `fubar` (similar to `git log foo bar ^fu ^fubar`).  Because the response could include a very large number of commits, it is paginated. Follow the 'next' link in the response to navigate to the next page of commits. As with other paginated resources, do not construct your own links.  When the include and exclude parameters are more than can fit in a query string, clients can use a `x-www-form-urlencoded` POST instead.

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

# **RepositoriesUsernameRepoSlugCommitsPost**
> ModelError RepositoriesUsernameRepoSlugCommitsPost(ctx, username, repoSlug)


Identical to `GET /repositories/{username}/{repo_slug}/commits`, except that POST allows clients to place the include and exclude parameters in the request body to avoid URL length issues.  **Note that this resource does NOT support new commit creation.**

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

# **RepositoriesUsernameRepoSlugCommitsRevisionGet**
> ModelError RepositoriesUsernameRepoSlugCommitsRevisionGet(ctx, username, revision, repoSlug)


These are the repository's commits. They are paginated and returned in reverse chronological order, similar to the output of `git log` and `hg log`. Like these tools, the DAG can be filtered.  ## GET /repositories/{username}/{repo_slug}/commits/  Returns all commits in the repo in topological order (newest commit first). All branches and tags are included (similar to `git log --all` and `hg log`).  ## GET /repositories/{username}/{repo_slug}/commits/master  Returns all commits on rev `master` (similar to `git log master`, `hg log master`).  ## GET /repositories/{username}/{repo_slug}/commits/dev?exclude=master  Returns all commits on ref `dev`, except those that are reachable on `master` (similar to `git log dev ^master`).  ## GET /repositories/{username}/{repo_slug}/commits/?exclude=master  Returns all commits in the repo that are not on master (similar to `git log --all ^master`).  ## GET /repositories/{username}/{repo_slug}/commits/?include=foo&include=bar&exclude=fu&exclude=fubar  Returns all commits that are on refs `foo` or `bar`, but not on `fu` or `fubar` (similar to `git log foo bar ^fu ^fubar`).  Because the response could include a very large number of commits, it is paginated. Follow the 'next' link in the response to navigate to the next page of commits. As with other paginated resources, do not construct your own links.  When the include and exclude parameters are more than can fit in a query string, clients can use a `x-www-form-urlencoded` POST instead.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **revision** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugCommitsRevisionPost**
> ModelError RepositoriesUsernameRepoSlugCommitsRevisionPost(ctx, username, revision, repoSlug)


Identical to `GET /repositories/{username}/{repo_slug}/commits`, except that POST allows clients to place the include and exclude parameters in the request body to avoid URL length issues.  **Note that this resource does NOT support new commit creation.**

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **revision** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugDiffSpecGet**
> RepositoriesUsernameRepoSlugDiffSpecGet(ctx, username, spec, repoSlug, optional)


Produces a raw, git-style diff for either a single commit (diffed against its first parent), or a revspec of 2 commits (e.g. `3a8b42..9ff173` where the first commit represents the source and the second commit the destination).  In case of the latter (diffing a revspec), a 3-way diff, or merge diff, is computed. This shows the changes introduced by the left branch (`3a8b42` in our example) as compared againt the right branch (`9ff173`).  This is equivalent to merging the left branch into the right branch and then computing the diff of the merge commit against its first parent (the right branch). This follows the same behavior as pull requests that also show this style of 3-way, or merge diff.  While similar to patches, diffs:  * Don't have a commit header (username, commit message, etc) * Support the optional `path=foo/bar.py` query param to filter   the diff to just that one file diff  The raw diff is returned as-is, in whatever encoding the files in the repository use. It is not decoded into unicode. As such, the content-type is `text/plain`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **spec** | **string**|  | 
  **repoSlug** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**|  | 
 **spec** | **string**|  | 
 **repoSlug** | **string**|  | 
 **context** | **int32**| Generate diffs with &lt;n&gt; lines of context instead of the usual three | 
 **path** | **string**| Limit the diff to a particular file (this parameter can be repeated for multiple paths) | 
 **ignoreWhitespace** | **bool**| Generate diffs that ignore whitespace | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPatchSpecGet**
> RepositoriesUsernameRepoSlugPatchSpecGet(ctx, username, spec, repoSlug)


Produces a raw patch for a single commit (diffed against its first parent), or a patch-series for a revspec of 2 commits (e.g. `3a8b42..9ff173` where the first commit represents the source and the second commit the destination).  In case of the latter (diffing a revspec), a patch series is returned for the commits on the source branch (`3a8b42` and its ancestors in our example). For Mercurial, a single patch is returned that combines the changes of all commits on the source branch.  While similar to diffs, patches:  * Have a commit header (username, commit message, etc) * Do not support the `path=foo/bar.py` query parameter  The raw patch is returned as-is, in whatever encoding the files in the repository use. It is not decoded into unicode. As such, the content-type is `text/plain`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **spec** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

