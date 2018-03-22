# \ProjectsApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TeamsUsernameProjectsGet**](ProjectsApi.md#TeamsUsernameProjectsGet) | **Get** /teams/{username}/projects/ | 
[**TeamsUsernameProjectsPost**](ProjectsApi.md#TeamsUsernameProjectsPost) | **Post** /teams/{username}/projects/ | 
[**TeamsUsernameProjectsProjectKeyDelete**](ProjectsApi.md#TeamsUsernameProjectsProjectKeyDelete) | **Delete** /teams/{username}/projects/{project_key} | 
[**TeamsUsernameProjectsProjectKeyGet**](ProjectsApi.md#TeamsUsernameProjectsProjectKeyGet) | **Get** /teams/{username}/projects/{project_key} | 
[**TeamsUsernameProjectsProjectKeyPut**](ProjectsApi.md#TeamsUsernameProjectsProjectKeyPut) | **Put** /teams/{username}/projects/{project_key} | 


# **TeamsUsernameProjectsGet**
> PaginatedProjects TeamsUsernameProjectsGet(ctx, username)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team which owns the project. This can either be the &#x60;username&#x60; of the team or the &#x60;UUID&#x60; of the team (surrounded by curly-braces (&#x60;{}&#x60;)).  | 

### Return type

[**PaginatedProjects**](paginated_projects.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameProjectsPost**
> Project TeamsUsernameProjectsPost(ctx, username, body)


Creates a new project.  Note that the avatar has to be embedded as either a data-url or a URL to an external image as shown in the examples below:  ``` $ body=$(cat << EOF {     \"name\": \"Mars Project\",     \"key\": \"MARS\",     \"description\": \"Software for colonizing mars.\",     \"links\": {         \"avatar\": {             \"href\": \"data:image/gif;base64,R0lGODlhEAAQAMQAAORHHOVSKudfOulrSOp3WOyDZu6QdvCchPGolfO0o/...\"         }     },     \"is_private\": false } EOF ) $ curl -H \"Content-Type: application/json\" \\        -X POST \\        -d \"$body\" \\        https://api.bitbucket.org/2.0/teams/teams-in-space/projects/ | jq . {   // Serialized project document } ```  or even:  ``` $ body=$(cat << EOF {     \"name\": \"Mars Project\",     \"key\": \"MARS\",     \"description\": \"Software for colonizing mars.\",     \"links\": {         \"avatar\": {             \"href\": \"http://i.imgur.com/72tRx4w.gif\"         }     },     \"is_private\": false } EOF ) $ curl -H \"Content-Type: application/json\" \\        -X POST \\        -d \"$body\" \\        https://api.bitbucket.org/2.0/teams/teams-in-space/projects/ | jq . {   // Serialized project document } ```

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team which owns the project. This can either be the &#x60;username&#x60; of the team or the &#x60;UUID&#x60; of the team (surrounded by curly-braces (&#x60;{}&#x60;)).  | 
  **body** | [**Project**](Project.md)|  | 

### Return type

[**Project**](project.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameProjectsProjectKeyDelete**
> TeamsUsernameProjectsProjectKeyDelete(ctx, username, projectKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team which owns the project. This can either be the &#x60;username&#x60; of the team or the &#x60;UUID&#x60; of the team (surrounded by curly-braces (&#x60;{}&#x60;)).  | 
  **projectKey** | **string**| The project in question. This can either be the actual &#x60;key&#x60; assigned to the project or the &#x60;UUID&#x60; (surrounded by curly-braces (&#x60;{}&#x60;)).  | 

### Return type

 (empty response body)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameProjectsProjectKeyGet**
> Project TeamsUsernameProjectsProjectKeyGet(ctx, username, projectKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team which owns the project. This can either be the &#x60;username&#x60; of the team or the &#x60;UUID&#x60; of the team (surrounded by curly-braces (&#x60;{}&#x60;)).  | 
  **projectKey** | **string**| The project in question. This can either be the actual &#x60;key&#x60; assigned to the project or the &#x60;UUID&#x60; (surrounded by curly-braces (&#x60;{}&#x60;)).  | 

### Return type

[**Project**](project.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameProjectsProjectKeyPut**
> Project TeamsUsernameProjectsProjectKeyPut(ctx, username, projectKey, body)


Since this endpoint can be used to both update and to create a project, the request body depends on the intent.  ### Creation  See the POST documentation for the project collection for an example of the request body.  Note: The `key` should not be specified in the body of request (since it is already present in the URL). The `name` is required, everything else is optional.  ### Update  See the POST documentation for the project collection for an example of the request body.  Note: The key is not required in the body (since it is already in the URL). The key may be specified in the body, if the intent is to change the key itself. In such a scenario, the location of the project is changed and is returned in the `Location` header of the response.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The team which owns the project. This can either be the &#x60;username&#x60; of the team or the &#x60;UUID&#x60; of the team (surrounded by curly-braces (&#x60;{}&#x60;)).  | 
  **projectKey** | **string**| The project in question. This can either be the actual &#x60;key&#x60; assigned to the project or the &#x60;UUID&#x60; (surrounded by curly-braces (&#x60;{}&#x60;)).  | 
  **body** | [**Project**](Project.md)|  | 

### Return type

[**Project**](project.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

