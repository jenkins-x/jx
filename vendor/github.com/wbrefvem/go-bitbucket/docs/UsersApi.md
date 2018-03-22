# \UsersApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TeamsUsernameRepositoriesGet**](UsersApi.md#TeamsUsernameRepositoriesGet) | **Get** /teams/{username}/repositories | 
[**UserEmailsEmailGet**](UsersApi.md#UserEmailsEmailGet) | **Get** /user/emails/{email} | 
[**UserEmailsGet**](UsersApi.md#UserEmailsGet) | **Get** /user/emails | 
[**UserGet**](UsersApi.md#UserGet) | **Get** /user | 
[**UsersUsernameFollowersGet**](UsersApi.md#UsersUsernameFollowersGet) | **Get** /users/{username}/followers | 
[**UsersUsernameFollowingGet**](UsersApi.md#UsersUsernameFollowingGet) | **Get** /users/{username}/following | 
[**UsersUsernameGet**](UsersApi.md#UsersUsernameGet) | **Get** /users/{username} | 
[**UsersUsernameHooksGet**](UsersApi.md#UsersUsernameHooksGet) | **Get** /users/{username}/hooks | 
[**UsersUsernameHooksPost**](UsersApi.md#UsersUsernameHooksPost) | **Post** /users/{username}/hooks | 
[**UsersUsernameHooksUidDelete**](UsersApi.md#UsersUsernameHooksUidDelete) | **Delete** /users/{username}/hooks/{uid} | 
[**UsersUsernameHooksUidGet**](UsersApi.md#UsersUsernameHooksUidGet) | **Get** /users/{username}/hooks/{uid} | 
[**UsersUsernameHooksUidPut**](UsersApi.md#UsersUsernameHooksUidPut) | **Put** /users/{username}/hooks/{uid} | 
[**UsersUsernameRepositoriesGet**](UsersApi.md#UsersUsernameRepositoriesGet) | **Get** /users/{username}/repositories | 


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

# **UserEmailsEmailGet**
> ModelError UserEmailsEmailGet(ctx, email)


Returns details about a specific one of the authenticated user's email addresses.  Details describe whether the address has been confirmed by the user and whether it is the user's primary address or not.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **email** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserEmailsGet**
> ModelError UserEmailsGet(ctx, )


Returns all the authenticated user's email addresses. Both confirmed and unconfirmed.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserGet**
> User UserGet(ctx, )


Returns the currently logged in user.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**User**](user.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameFollowersGet**
> PaginatedUsers UsersUsernameFollowersGet(ctx, username)


Returns the list of accounts that are following this team.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account&#39;s username | 

### Return type

[**PaginatedUsers**](paginated_users.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameFollowingGet**
> PaginatedUsers UsersUsernameFollowingGet(ctx, username)


Returns the list of accounts this user is following.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The user&#39;s username | 

### Return type

[**PaginatedUsers**](paginated_users.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameGet**
> User UsersUsernameGet(ctx, username)


Gets the public information associated with a user account.  If the user's profile is private, `location`, `website` and `created_on` elements are omitted.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account&#39;s username or UUID. | 

### Return type

[**User**](user.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameHooksGet**
> PaginatedWebhookSubscriptions UsersUsernameHooksGet(ctx, username)


Returns a paginated list of webhooks installed on this user account.

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

# **UsersUsernameHooksPost**
> WebhookSubscription UsersUsernameHooksPost(ctx, username)


Creates a new webhook on the specified user account.  Account-level webhooks are fired for events from all repositories belonging to that account.  Note that one can only register webhooks on one's own account, not that of others.

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

# **UsersUsernameHooksUidDelete**
> UsersUsernameHooksUidDelete(ctx, username, uid)


Deletes the specified webhook subscription from the given user account.

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

# **UsersUsernameHooksUidGet**
> WebhookSubscription UsersUsernameHooksUidGet(ctx, username, uid)


Returns the webhook with the specified id installed on the given user account.

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

# **UsersUsernameHooksUidPut**
> WebhookSubscription UsersUsernameHooksUidPut(ctx, username, uid)


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

