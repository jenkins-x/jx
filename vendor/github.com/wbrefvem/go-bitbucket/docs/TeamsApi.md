# \TeamsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TeamsGet**](TeamsApi.md#TeamsGet) | **Get** /teams | 
[**TeamsUsernameFollowersGet**](TeamsApi.md#TeamsUsernameFollowersGet) | **Get** /teams/{username}/followers | 
[**TeamsUsernameFollowingGet**](TeamsApi.md#TeamsUsernameFollowingGet) | **Get** /teams/{username}/following | 
[**TeamsUsernameGet**](TeamsApi.md#TeamsUsernameGet) | **Get** /teams/{username} | 
[**TeamsUsernameHooksGet**](TeamsApi.md#TeamsUsernameHooksGet) | **Get** /teams/{username}/hooks | 
[**TeamsUsernameHooksPost**](TeamsApi.md#TeamsUsernameHooksPost) | **Post** /teams/{username}/hooks | 
[**TeamsUsernameHooksUidDelete**](TeamsApi.md#TeamsUsernameHooksUidDelete) | **Delete** /teams/{username}/hooks/{uid} | 
[**TeamsUsernameHooksUidGet**](TeamsApi.md#TeamsUsernameHooksUidGet) | **Get** /teams/{username}/hooks/{uid} | 
[**TeamsUsernameHooksUidPut**](TeamsApi.md#TeamsUsernameHooksUidPut) | **Put** /teams/{username}/hooks/{uid} | 
[**TeamsUsernameMembersGet**](TeamsApi.md#TeamsUsernameMembersGet) | **Get** /teams/{username}/members | 
[**TeamsUsernameRepositoriesGet**](TeamsApi.md#TeamsUsernameRepositoriesGet) | **Get** /teams/{username}/repositories | 
[**UsersUsernameRepositoriesGet**](TeamsApi.md#UsersUsernameRepositoriesGet) | **Get** /users/{username}/repositories | 


# **TeamsGet**
> PaginatedTeams TeamsGet(ctx, optional)


Returns all the teams that the authenticated user is associated with.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **role** | **string**|  Filters the teams based on the authenticated user&#39;s role on each team.  * **member**: returns a list of all the teams which the caller is a member of   at least one team group or repository owned by the team * **contributor**: returns a list of teams which the caller has write access   to at least one repository owned by the team * **admin**: returns a list teams which the caller has team administrator access  | 

### Return type

[**PaginatedTeams**](paginated_teams.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameFollowersGet**
> PaginatedUsers TeamsUsernameFollowersGet(ctx, username)


Returns the list of accounts that are following this team.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team&#39;s username | 

### Return type

[**PaginatedUsers**](paginated_users.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameFollowingGet**
> PaginatedUsers TeamsUsernameFollowingGet(ctx, username)


Returns the list of accounts this team is following.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team&#39;s username | 

### Return type

[**PaginatedUsers**](paginated_users.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameGet**
> Team TeamsUsernameGet(ctx, username)


Gets the public information associated with a team.  If the team's profile is private, `location`, `website` and `created_on` elements are omitted.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team&#39;s username or UUID. | 

### Return type

[**Team**](team.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameHooksGet**
> PaginatedWebhookSubscriptions TeamsUsernameHooksGet(ctx, username)


Returns a paginated list of webhooks installed on this team.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 

### Return type

[**PaginatedWebhookSubscriptions**](paginated_webhook_subscriptions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameHooksPost**
> WebhookSubscription TeamsUsernameHooksPost(ctx, username)


Creates a new webhook on the specified team.  Team webhooks are fired for events from all repositories belonging to that team account.  Note that only admins can install webhooks on teams.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameHooksUidDelete**
> TeamsUsernameHooksUidDelete(ctx, username, uid)


Deletes the specified webhook subscription from the given team account.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameHooksUidGet**
> WebhookSubscription TeamsUsernameHooksUidGet(ctx, username, uid)


Returns the webhook with the specified id installed on the given team account.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id. | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameHooksUidPut**
> WebhookSubscription TeamsUsernameHooksUidPut(ctx, username, uid)


Updates the specified webhook subscription.  The following properties can be mutated:  * `description` * `url` * `active` * `events`

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameMembersGet**
> User TeamsUsernameMembersGet(ctx, username)


All members of a team.  Returns all members of the specified team. Any member of any of the team's groups is considered a member of the team. This includes users in groups that may not actually have access to any of the team's repositories.  Note that members using the \"private profile\" feature are not included.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 

### Return type

[**User**](user.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameRepositoriesGet**
> ModelError TeamsUsernameRepositoriesGet(ctx, username)


All repositories owned by a user/team. This includes private repositories, but filtered down to the ones that the calling user has access to.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameRepositoriesGet**
> ModelError UsersUsernameRepositoriesGet(ctx, username)


All repositories owned by a user/team. This includes private repositories, but filtered down to the ones that the calling user has access to.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

