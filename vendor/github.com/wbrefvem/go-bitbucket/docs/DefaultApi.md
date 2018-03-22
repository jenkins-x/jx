# \DefaultApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameDelete**](DefaultApi.md#RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameDelete) | **Delete** /repositories/{username}/{repo_slug}/properties/{app_key}/{property_name} | 
[**RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameGet**](DefaultApi.md#RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameGet) | **Get** /repositories/{username}/{repo_slug}/properties/{app_key}/{property_name} | 
[**RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNamePut**](DefaultApi.md#RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNamePut) | **Put** /repositories/{username}/{repo_slug}/properties/{app_key}/{property_name} | 
[**TeamsUsernamePermissionsGet**](DefaultApi.md#TeamsUsernamePermissionsGet) | **Get** /teams/{username}/permissions | 
[**TeamsUsernamePermissionsRepositoriesGet**](DefaultApi.md#TeamsUsernamePermissionsRepositoriesGet) | **Get** /teams/{username}/permissions/repositories | 
[**TeamsUsernameSearchCodeGet**](DefaultApi.md#TeamsUsernameSearchCodeGet) | **Get** /teams/{username}/search/code | 
[**UserPermissionsTeamsGet**](DefaultApi.md#UserPermissionsTeamsGet) | **Get** /user/permissions/teams | 
[**UsersUsernameSearchCodeGet**](DefaultApi.md#UsersUsernameSearchCodeGet) | **Get** /users/{username}/search/code | 


# **RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameDelete**
> RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameDelete(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameGet**
> RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNameGet(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNamePut**
> RepositoriesUsernameRepoSlugPropertiesAppKeyPropertyNamePut(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernamePermissionsGet**
> PaginatedTeamPermissions TeamsUsernamePermissionsGet(ctx, username, optional)


Returns an object for each team permission a user on the team has.  Permissions returned are effective permissions — if a user is a member of multiple groups with distinct roles, only the highest level is returned.  Permissions can be:  * `admin` * `collaborator`  Only users with admin permission for the team may access this resource.  Example:  ``` $ curl https://api.bitbucket.org/2.0/teams/atlassian_tutorial/permissions  {   \"pagelen\": 10,   \"values\": [     {       \"permission\": \"admin\",       \"type\": \"team_permission\",       \"user\": {         \"type\": \"user\",         \"username\": \"evzijst\",         \"display_name\": \"Erik van Zijst\",         \"uuid\": \"{d301aafa-d676-4ee0-88be-962be7417567}\"       },       \"team\": {         \"username\": \"bitbucket\",         \"display_name\": \"Atlassian Bitbucket\",         \"uuid\": \"{4cc6108a-a241-4db0-96a5-64347ac04f87}\"       }     },     {       \"permission\": \"collaborator\",       \"type\": \"team_permission\",       \"user\": {         \"type\": \"user\",         \"username\": \"seanaty\",         \"display_name\": \"Sean Conaty\",         \"uuid\": \"{504c3b62-8120-4f0c-a7bc-87800b9d6f70}\"       },       \"team\": {         \"username\": \"bitbucket\",         \"display_name\": \"Atlassian Bitbucket\",         \"uuid\": \"{4cc6108a-a241-4db0-96a5-64347ac04f87}\"       }     }   ],   \"page\": 1,   \"size\": 2 } ```  Results may be further [filtered or sorted](../../../meta/filtering) by team, user, or permission by adding the following query string parameters:  * `q=user.username=\"evzijst\"` or `q=permission=\"admin\"` * `sort=team.display_name`  Note that the query parameter values need to be URL escaped so that `=` would become `%3D`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the team, surrounded by curly-braces, for example: &#x60;{team UUID}&#x60;.  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| This can either be the username or the UUID of the team, surrounded by curly-braces, for example: &#x60;{team UUID}&#x60;.  | 
 **q** | **string**|  Query string to narrow down the response as per [filtering and sorting](../../../meta/filtering). | 
 **sort** | **string**|  Name of a response property sort the result by as per [filtering and sorting](../../../meta/filtering#query-sort).  | 

### Return type

[**PaginatedTeamPermissions**](paginated_team_permissions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernamePermissionsRepositoriesGet**
> PaginatedRepositoryPermissions TeamsUsernamePermissionsRepositoriesGet(ctx, username, optional)


Returns an object for each repository permission for all of a team’s repositories.  Permissions returned are effective permissions — the highest level of permission the user has. This does not include public repositories that users are not granted any specific permission in, and does not distinguish between direct and indirect privileges.  Only users with admin permission for the team may access this resource.  Permissions can be:  * `admin` * `write` * `read`  Example:  ``` $ curl https://api.bitbucket.org/2.0/teams/atlassian_tutorial/permissions/repositories  {   \"pagelen\": 10,   \"values\": [     {       \"type\": \"repository_permission\",       \"user\": {         \"type\": \"user\",         \"username\": \"evzijst\",         \"display_name\": \"Erik van Zijst\",         \"uuid\": \"{d301aafa-d676-4ee0-88be-962be7417567}\"       },       \"repository\": {         \"type\": \"repository\",         \"name\": \"geordi\",         \"full_name\": \"bitbucket/geordi\",         \"uuid\": \"{85d08b4e-571d-44e9-a507-fa476535aa98}\"       },       \"permission\": \"admin\"     },     {       \"type\": \"repository_permission\",       \"user\": {         \"type\": \"user\",         \"username\": \"seanaty\",         \"display_name\": \"Sean Conaty\",         \"uuid\": \"{504c3b62-8120-4f0c-a7bc-87800b9d6f70}\"       },       \"repository\": {         \"type\": \"repository\",         \"name\": \"geordi\",         \"full_name\": \"bitbucket/geordi\",         \"uuid\": \"{85d08b4e-571d-44e9-a507-fa476535aa98}\"       },       \"permission\": \"write\"     }   ],   \"page\": 1,   \"size\": 2 } ```  Results may be further [filtered or sorted](../../../../meta/filtering) by repository, user, or permission by adding the following query string parameters:  * `q=repository.name=\"geordi\"` or `q=permission>\"read\"` * `sort=user.display_name`  Note that the query parameter values need to be URL escaped so that `=` would become `%3D`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| This can either be the username or the UUID of the team, surrounded by curly-braces, for example: &#x60;{team UUID}&#x60;.  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| This can either be the username or the UUID of the team, surrounded by curly-braces, for example: &#x60;{team UUID}&#x60;.  | 
 **q** | **string**|  Query string to narrow down the response as per [filtering and sorting](../../../../meta/filtering). | 
 **sort** | **string**|  Name of a response property sort the result by as per [filtering and sorting](../../../../meta/filtering#query-sort).  | 

### Return type

[**PaginatedRepositoryPermissions**](paginated_repository_permissions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TeamsUsernameSearchCodeGet**
> SearchResult TeamsUsernameSearchCodeGet(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

[**SearchResult**](search_result.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserPermissionsTeamsGet**
> PaginatedTeamPermissions UserPermissionsTeamsGet(ctx, optional)


Returns an object for each team the caller is a member of, and their effective role — the highest level of privilege the caller has. If a user is a member of multiple groups with distinct roles, only the highest level is returned.  Permissions can be:  * `admin` * `collaborator`  Example:  ``` $ curl https://api.bitbucket.org/2.0/user/permissions/teams  {   \"pagelen\": 10,   \"values\": [     {       \"permission\": \"admin\",       \"type\": \"team_permission\",       \"user\": {         \"type\": \"user\",         \"username\": \"evzijst\",         \"display_name\": \"Erik van Zijst\",         \"uuid\": \"{d301aafa-d676-4ee0-88be-962be7417567}\"       },       \"team\": {         \"username\": \"bitbucket\",         \"display_name\": \"Atlassian Bitbucket\",         \"uuid\": \"{4cc6108a-a241-4db0-96a5-64347ac04f87}\"       }     }   ],   \"page\": 1,   \"size\": 1 } ```  Results may be further [filtered or sorted](../../../meta/filtering) by team or permission by adding the following query string parameters:  * `q=team.username=\"bitbucket\"` or `q=permission=\"admin\"` * `sort=team.display_name`  Note that the query parameter values need to be URL escaped so that `=` would become `%3D`.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **q** | **string**|  Query string to narrow down the response as per [filtering and sorting](../../../meta/filtering). | 
 **sort** | **string**|  Name of a response property sort the result by as per [filtering and sorting](../../../meta/filtering#query-sort).  | 

### Return type

[**PaginatedTeamPermissions**](paginated_team_permissions.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UsersUsernameSearchCodeGet**
> SearchResult UsersUsernameSearchCodeGet(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

[**SearchResult**](search_result.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

