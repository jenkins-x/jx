# \WebhooksApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**HookEventsGet**](WebhooksApi.md#HookEventsGet) | **Get** /hook_events | 
[**HookEventsSubjectTypeGet**](WebhooksApi.md#HookEventsSubjectTypeGet) | **Get** /hook_events/{subject_type} | 
[**RepositoriesUsernameRepoSlugHooksGet**](WebhooksApi.md#RepositoriesUsernameRepoSlugHooksGet) | **Get** /repositories/{username}/{repo_slug}/hooks | 
[**RepositoriesUsernameRepoSlugHooksPost**](WebhooksApi.md#RepositoriesUsernameRepoSlugHooksPost) | **Post** /repositories/{username}/{repo_slug}/hooks | 
[**RepositoriesUsernameRepoSlugHooksUidDelete**](WebhooksApi.md#RepositoriesUsernameRepoSlugHooksUidDelete) | **Delete** /repositories/{username}/{repo_slug}/hooks/{uid} | 
[**RepositoriesUsernameRepoSlugHooksUidGet**](WebhooksApi.md#RepositoriesUsernameRepoSlugHooksUidGet) | **Get** /repositories/{username}/{repo_slug}/hooks/{uid} | 
[**RepositoriesUsernameRepoSlugHooksUidPut**](WebhooksApi.md#RepositoriesUsernameRepoSlugHooksUidPut) | **Put** /repositories/{username}/{repo_slug}/hooks/{uid} | 
[**TeamsUsernameHooksGet**](WebhooksApi.md#TeamsUsernameHooksGet) | **Get** /teams/{username}/hooks | 
[**TeamsUsernameHooksPost**](WebhooksApi.md#TeamsUsernameHooksPost) | **Post** /teams/{username}/hooks | 
[**TeamsUsernameHooksUidDelete**](WebhooksApi.md#TeamsUsernameHooksUidDelete) | **Delete** /teams/{username}/hooks/{uid} | 
[**TeamsUsernameHooksUidGet**](WebhooksApi.md#TeamsUsernameHooksUidGet) | **Get** /teams/{username}/hooks/{uid} | 
[**TeamsUsernameHooksUidPut**](WebhooksApi.md#TeamsUsernameHooksUidPut) | **Put** /teams/{username}/hooks/{uid} | 
[**UsersUsernameHooksGet**](WebhooksApi.md#UsersUsernameHooksGet) | **Get** /users/{username}/hooks | 
[**UsersUsernameHooksPost**](WebhooksApi.md#UsersUsernameHooksPost) | **Post** /users/{username}/hooks | 
[**UsersUsernameHooksUidDelete**](WebhooksApi.md#UsersUsernameHooksUidDelete) | **Delete** /users/{username}/hooks/{uid} | 
[**UsersUsernameHooksUidGet**](WebhooksApi.md#UsersUsernameHooksUidGet) | **Get** /users/{username}/hooks/{uid} | 
[**UsersUsernameHooksUidPut**](WebhooksApi.md#UsersUsernameHooksUidPut) | **Put** /users/{username}/hooks/{uid} | 


# **HookEventsGet**
> SubjectTypes HookEventsGet(ctx, )


Returns the webhook resource or subject types on which webhooks can be registered.  Each resource/subject type contains an `events` link that returns the paginated list of specific events each individual subject type can emit.  This endpoint is publicly accessible and does not require authentication or scopes.  Example:  ``` $ curl https://api.bitbucket.org/2.0/hook_events  {     \"repository\": {         \"links\": {             \"events\": {                 \"href\": \"https://api.bitbucket.org/2.0/hook_events/repository\"             }         }     },     \"team\": {         \"links\": {             \"events\": {                 \"href\": \"https://api.bitbucket.org/2.0/hook_events/team\"             }         }     },     \"user\": {         \"links\": {             \"events\": {                 \"href\": \"https://api.bitbucket.org/2.0/hook_events/user\"             }         }     } } ```

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**SubjectTypes**](subject_types.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **HookEventsSubjectTypeGet**
> PaginatedHookEvents HookEventsSubjectTypeGet(ctx, subjectType)


Returns a paginated list of all valid webhook events for the specified entity.  This is public data that does not require any scopes or authentication.  Example:  NOTE: The following example is a truncated response object for the `team` `subject_type`. We return the same structure for the other `subject_type` objects.  ``` $ curl https://api.bitbucket.org/2.0/hook_events/team {     \"page\": 1,     \"pagelen\": 30,     \"size\": 21,     \"values\": [         {             \"category\": \"Repository\",             \"description\": \"Whenever a repository push occurs\",             \"event\": \"repo:push\",             \"label\": \"Push\"         },         {             \"category\": \"Repository\",             \"description\": \"Whenever a repository fork occurs\",             \"event\": \"repo:fork\",             \"label\": \"Fork\"         },         ...         {             \"category\": \"Repository\",             \"description\": \"Whenever a repository import occurs\",             \"event\": \"repo:imported\",             \"label\": \"Import\"         }     ] } ```

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **subjectType** | **string**| A resource or subject type. | 

### Return type

[**PaginatedHookEvents**](paginated_hook_events.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugHooksGet**
> PaginatedWebhookSubscriptions RepositoriesUsernameRepoSlugHooksGet(ctx, username, repoSlug)


Returns a paginated list of webhooks installed on this repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**PaginatedWebhookSubscriptions**](paginated_webhook_subscriptions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugHooksPost**
> WebhookSubscription RepositoriesUsernameRepoSlugHooksPost(ctx, username, repoSlug)


Creates a new webhook on the specified repository.  Example:  ``` $ curl -X POST -u credentials -H 'Content-Type: application/json'           https://api.bitbucket.org/2.0/repositories/username/slug/hooks           -d '     {       \"description\": \"Webhook Description\",       \"url\": \"https://example.com/\",       \"active\": true,       \"events\": [         \"repo:push\",         \"issue:created\",         \"issue:updated\"       ]     }' ```  Note that this call requires the webhook scope, as well as any scope that applies to the events that the webhook subscribes to. In the example above that means: `webhook`, `repository` and `issue`.  Also note that the `url` must properly resolve and cannot be an internal, non-routed address.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugHooksUidDelete**
> RepositoriesUsernameRepoSlugHooksUidDelete(ctx, username, repoSlug, uid)


Deletes the specified webhook subscription from the given repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugHooksUidGet**
> WebhookSubscription RepositoriesUsernameRepoSlugHooksUidGet(ctx, username, repoSlug, uid)


Returns the webhook with the specified id installed on the specified repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id. | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugHooksUidPut**
> WebhookSubscription RepositoriesUsernameRepoSlugHooksUidPut(ctx, username, repoSlug, uid)


Updates the specified webhook subscription.  The following properties can be mutated:  * `description` * `url` * `active` * `events`

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**|  | 
  **repoSlug** | **string**|  | 
  **uid** | **string**| The installed webhook&#39;s id | 

### Return type

[**WebhookSubscription**](webhook_subscription.md)

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

