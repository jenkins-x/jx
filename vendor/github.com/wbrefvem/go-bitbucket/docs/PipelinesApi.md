# \PipelinesApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreatePipelineForRepository**](PipelinesApi.md#CreatePipelineForRepository) | **Post** /repositories/{username}/{repo_slug}/pipelines/ | 
[**CreatePipelineVariableForTeam**](PipelinesApi.md#CreatePipelineVariableForTeam) | **Post** /teams/{username}/pipelines_config/variables/ | 
[**CreatePipelineVariableForUser**](PipelinesApi.md#CreatePipelineVariableForUser) | **Post** /users/{username}/pipelines_config/variables/ | 
[**CreateRepositoryPipelineKnownHost**](PipelinesApi.md#CreateRepositoryPipelineKnownHost) | **Post** /repositories/{username}/{repo_slug}/pipelines_config/ssh/known_hosts/ | 
[**CreateRepositoryPipelineSchedule**](PipelinesApi.md#CreateRepositoryPipelineSchedule) | **Post** /repositories/{username}/{repo_slug}/pipelines_config/schedules/ | 
[**CreateRepositoryPipelineVariable**](PipelinesApi.md#CreateRepositoryPipelineVariable) | **Post** /repositories/{username}/{repo_slug}/pipelines_config/variables/ | 
[**DeletePipelineVariableForTeam**](PipelinesApi.md#DeletePipelineVariableForTeam) | **Delete** /teams/{username}/pipelines_config/variables/{variable_uuid} | 
[**DeletePipelineVariableForUser**](PipelinesApi.md#DeletePipelineVariableForUser) | **Delete** /users/{username}/pipelines_config/variables/{variable_uuid} | 
[**DeleteRepositoryPipelineKeyPair**](PipelinesApi.md#DeleteRepositoryPipelineKeyPair) | **Delete** /repositories/{username}/{repo_slug}/pipelines_config/ssh/key_pair | 
[**DeleteRepositoryPipelineKnownHost**](PipelinesApi.md#DeleteRepositoryPipelineKnownHost) | **Delete** /repositories/{username}/{repo_slug}/pipelines_config/ssh/known_hosts/{known_host_uuid} | 
[**DeleteRepositoryPipelineSchedule**](PipelinesApi.md#DeleteRepositoryPipelineSchedule) | **Delete** /repositories/{username}/{repo_slug}/pipelines_config/schedules/{schedule_uuid} | 
[**DeleteRepositoryPipelineVariable**](PipelinesApi.md#DeleteRepositoryPipelineVariable) | **Delete** /repositories/{username}/{repo_slug}/pipelines_config/variables/{variable_uuid} | 
[**GetPipelineForRepository**](PipelinesApi.md#GetPipelineForRepository) | **Get** /repositories/{username}/{repo_slug}/pipelines/{pipeline_uuid} | 
[**GetPipelineStepForRepository**](PipelinesApi.md#GetPipelineStepForRepository) | **Get** /repositories/{username}/{repo_slug}/pipelines/{pipeline_uuid}/steps/{step_uuid} | 
[**GetPipelineStepLogForRepository**](PipelinesApi.md#GetPipelineStepLogForRepository) | **Get** /repositories/{username}/{repo_slug}/pipelines/{pipeline_uuid}/steps/{step_uuid}/log | 
[**GetPipelineStepsForRepository**](PipelinesApi.md#GetPipelineStepsForRepository) | **Get** /repositories/{username}/{repo_slug}/pipelines/{pipeline_uuid}/steps/ | 
[**GetPipelineVariableForTeam**](PipelinesApi.md#GetPipelineVariableForTeam) | **Get** /teams/{username}/pipelines_config/variables/{variable_uuid} | 
[**GetPipelineVariableForUser**](PipelinesApi.md#GetPipelineVariableForUser) | **Get** /users/{username}/pipelines_config/variables/{variable_uuid} | 
[**GetPipelineVariablesForTeam**](PipelinesApi.md#GetPipelineVariablesForTeam) | **Get** /teams/{username}/pipelines_config/variables/ | 
[**GetPipelineVariablesForUser**](PipelinesApi.md#GetPipelineVariablesForUser) | **Get** /users/{username}/pipelines_config/variables/ | 
[**GetPipelinesForRepository**](PipelinesApi.md#GetPipelinesForRepository) | **Get** /repositories/{username}/{repo_slug}/pipelines/ | 
[**GetRepositoryPipelineConfig**](PipelinesApi.md#GetRepositoryPipelineConfig) | **Get** /repositories/{username}/{repo_slug}/pipelines_config | 
[**GetRepositoryPipelineKnownHost**](PipelinesApi.md#GetRepositoryPipelineKnownHost) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/ssh/known_hosts/{known_host_uuid} | 
[**GetRepositoryPipelineKnownHosts**](PipelinesApi.md#GetRepositoryPipelineKnownHosts) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/ssh/known_hosts/ | 
[**GetRepositoryPipelineSchedule**](PipelinesApi.md#GetRepositoryPipelineSchedule) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/schedules/{schedule_uuid} | 
[**GetRepositoryPipelineScheduleExecutions**](PipelinesApi.md#GetRepositoryPipelineScheduleExecutions) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/schedules/{schedule_uuid}/executions/ | 
[**GetRepositoryPipelineSchedules**](PipelinesApi.md#GetRepositoryPipelineSchedules) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/schedules/ | 
[**GetRepositoryPipelineSshKeyPair**](PipelinesApi.md#GetRepositoryPipelineSshKeyPair) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/ssh/key_pair | 
[**GetRepositoryPipelineVariable**](PipelinesApi.md#GetRepositoryPipelineVariable) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/variables/{variable_uuid} | 
[**GetRepositoryPipelineVariables**](PipelinesApi.md#GetRepositoryPipelineVariables) | **Get** /repositories/{username}/{repo_slug}/pipelines_config/variables/ | 
[**StopPipeline**](PipelinesApi.md#StopPipeline) | **Post** /repositories/{username}/{repo_slug}/pipelines/{pipeline_uuid}/stopPipeline | 
[**UpdatePipelineVariableForTeam**](PipelinesApi.md#UpdatePipelineVariableForTeam) | **Put** /teams/{username}/pipelines_config/variables/{variable_uuid} | 
[**UpdatePipelineVariableForUser**](PipelinesApi.md#UpdatePipelineVariableForUser) | **Put** /users/{username}/pipelines_config/variables/{variable_uuid} | 
[**UpdateRepositoryBuildNumber**](PipelinesApi.md#UpdateRepositoryBuildNumber) | **Put** /repositories/{username}/{repo_slug}/pipelines_config/build_number | 
[**UpdateRepositoryPipelineConfig**](PipelinesApi.md#UpdateRepositoryPipelineConfig) | **Put** /repositories/{username}/{repo_slug}/pipelines_config | 
[**UpdateRepositoryPipelineKeyPair**](PipelinesApi.md#UpdateRepositoryPipelineKeyPair) | **Put** /repositories/{username}/{repo_slug}/pipelines_config/ssh/key_pair | 
[**UpdateRepositoryPipelineKnownHost**](PipelinesApi.md#UpdateRepositoryPipelineKnownHost) | **Put** /repositories/{username}/{repo_slug}/pipelines_config/ssh/known_hosts/{known_host_uuid} | 
[**UpdateRepositoryPipelineSchedule**](PipelinesApi.md#UpdateRepositoryPipelineSchedule) | **Put** /repositories/{username}/{repo_slug}/pipelines_config/schedules/{schedule_uuid} | 
[**UpdateRepositoryPipelineVariable**](PipelinesApi.md#UpdateRepositoryPipelineVariable) | **Put** /repositories/{username}/{repo_slug}/pipelines_config/variables/{variable_uuid} | 


# **CreatePipelineForRepository**
> Pipeline CreatePipelineForRepository(ctx, username, repoSlug, body)


Endpoint to create and initiate a pipeline.  There are a couple of different options to initiate a pipeline, where the payload of the request will determine which type of pipeline will be instantiated. # Trigger a Pipeline for a branch or tag One way to trigger pipelines is by specifying the reference for which you want to trigger a pipeline (e.g. a branch or tag).  The specified reference will be used to determine which pipeline definition from the `bitbucket-pipelines.yml` file will be applied to initiate the pipeline. The pipeline will then do a clone of the repository and checkout the latest revision of the specified reference.  ### Example  ``` $ curl -X POST -is -u username:password \\   -H 'Content-Type: application/json' \\  https://api.bitbucket.org/2.0/repositories/jeroendr/meat-demo2/pipelines/ \\   -d '   {     \"target\": {       \"ref_type\": \"branch\",        \"type\": \"pipeline_ref_target\",        \"ref_name\": \"master\"     }   }' ``` # Trigger a Pipeline for a commit on a branch or tag You can initiate a pipeline for a specific commit and in the context of a specified reference (e.g. a branch, tag or bookmark). The specified reference will be used to determine which pipeline definition from the bitbucket-pipelines.yml file will be applied to initiate the pipeline. The pipeline will clone the repository and then do a checkout the specified reference.   The following reference types are supported:  * `branch`  * `named_branch` * `bookmark`   * `tag`  ### Example  ``` $ curl -X POST -is -u username:password \\   -H 'Content-Type: application/json' \\   https://api.bitbucket.org/2.0/repositories/jeroendr/meat-demo2/pipelines/ \\   -d '   {     \"target\": {       \"commit\": {         \"type\": \"commit\",          \"hash\": \"ce5b7431602f7cbba007062eeb55225c6e18e956\"       },        \"ref_type\": \"branch\",        \"type\": \"pipeline_ref_target\",        \"ref_name\": \"master\"     }   }' ``` # Trigger a specific pipeline definition for a commit You can trigger a specific pipeline that is defined in your `bitbucket-pipelines.yml` file for a specific commit.  In addition to the commit revision, you specify the type and pattern of the selector that identifies the pipeline definition. The resulting pipeline will then clone the repository and checkout the specified revision.  ### Example  ``` $ curl -X POST -is -u username:password \\   -H 'Content-Type: application/json' \\  https://api.bitbucket.org/2.0/repositories/jeroendr/meat-demo2/pipelines/ \\  -d '   {      \"target\": {       \"commit\": {          \"hash\":\"a3c4e02c9a3755eccdc3764e6ea13facdf30f923\",          \"type\":\"commit\"        },         \"selector\": {            \"type\":\"custom\",               \"pattern\":\"Deploy to production\"           },         \"type\":\"pipeline_commit_target\"    }   }' ``` # Trigger a specific pipeline definition for a commit on a branch or tag You can trigger a specific pipeline that is defined in your `bitbucket-pipelines.yml` file for a specific commit in the context of a specified reference.  In addition to the commit revision, you specify the type and pattern of the selector that identifies the pipeline definition, as well as the reference information. The resulting pipeline will then clone the repository a checkout the specified reference.  ### Example  ``` $ curl -X POST -is -u username:password \\   -H 'Content-Type: application/json' \\  https://api.bitbucket.org/2.0/repositories/jeroendr/meat-demo2/pipelines/ \\  -d '   {      \"target\": {       \"commit\": {          \"hash\":\"a3c4e02c9a3755eccdc3764e6ea13facdf30f923\",          \"type\":\"commit\"        },        \"selector\": {           \"type\": \"custom\",           \"pattern\": \"Deploy to production\"        },        \"type\": \"pipeline_ref_target\",        \"ref_name\": \"master\",        \"ref_type\": \"branch\"      }   }' ``` 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**Pipeline**](Pipeline.md)| The pipeline to initiate. | 

### Return type

[**Pipeline**](pipeline.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreatePipelineVariableForTeam**
> PipelineVariable CreatePipelineVariableForTeam(ctx, username, optional)


Create an account level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| The account. | 
 **body** | [**PipelineVariable**](PipelineVariable.md)| The variable to create. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreatePipelineVariableForUser**
> PipelineVariable CreatePipelineVariableForUser(ctx, username, optional)


Create a user level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **username** | **string**| The account. | 
 **body** | [**PipelineVariable**](PipelineVariable.md)| The variable to create. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateRepositoryPipelineKnownHost**
> PipelineKnownHost CreateRepositoryPipelineKnownHost(ctx, username, repoSlug, body)


Create a repository level known host.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelineKnownHost**](PipelineKnownHost.md)| The known host to create. | 

### Return type

[**PipelineKnownHost**](pipeline_known_host.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateRepositoryPipelineSchedule**
> PipelineSchedule CreateRepositoryPipelineSchedule(ctx, username, repoSlug, body)


Create a schedule for the given repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelineSchedule**](PipelineSchedule.md)| The schedule to create. | 

### Return type

[**PipelineSchedule**](pipeline_schedule.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateRepositoryPipelineVariable**
> PipelineVariable CreateRepositoryPipelineVariable(ctx, username, repoSlug, body)


Create a repository level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelineVariable**](PipelineVariable.md)| The variable to create. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeletePipelineVariableForTeam**
> DeletePipelineVariableForTeam(ctx, username, variableUuid)


Delete a team level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable to delete. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeletePipelineVariableForUser**
> DeletePipelineVariableForUser(ctx, username, variableUuid)


Delete an account level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable to delete. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRepositoryPipelineKeyPair**
> DeleteRepositoryPipelineKeyPair(ctx, username, repoSlug)


Delete the repository SSH key pair.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRepositoryPipelineKnownHost**
> DeleteRepositoryPipelineKnownHost(ctx, username, repoSlug, knownHostUuid)


Delete a repository level known host.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **knownHostUuid** | **string**| The UUID of the known host to delete. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRepositoryPipelineSchedule**
> DeleteRepositoryPipelineSchedule(ctx, username, repoSlug, scheduleUuid)


Delete a schedule.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **scheduleUuid** | **string**| The uuid of the schedule. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRepositoryPipelineVariable**
> DeleteRepositoryPipelineVariable(ctx, username, repoSlug, variableUuid)


Delete a repository level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **variableUuid** | **string**| The UUID of the variable to delete. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineForRepository**
> Pipeline GetPipelineForRepository(ctx, username, repoSlug, pipelineUuid)


Retrieve a specified pipeline

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **pipelineUuid** | **string**| The pipeline UUID. | 

### Return type

[**Pipeline**](pipeline.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineStepForRepository**
> PipelineStep GetPipelineStepForRepository(ctx, username, repoSlug, pipelineUuid, stepUuid)


Retrieve a given step of a pipeline.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **pipelineUuid** | **string**| The UUID of the pipeline. | 
  **stepUuid** | **string**| The UUID of the step. | 

### Return type

[**PipelineStep**](pipeline_step.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineStepLogForRepository**
> GetPipelineStepLogForRepository(ctx, username, repoSlug, pipelineUuid, stepUuid)


Retrieve the log file for a given step of a pipeline.  This endpoint supports (and encourages!) the use of [HTTP Range requests](https://tools.ietf.org/html/rfc7233) to deal with potentially very large log files.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **pipelineUuid** | **string**| The UUID of the pipeline. | 
  **stepUuid** | **string**| The UUID of the step. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineStepsForRepository**
> PaginatedPipelineSteps GetPipelineStepsForRepository(ctx, username, repoSlug, pipelineUuid)


Find steps for the given pipeline.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **pipelineUuid** | **string**| The UUID of the pipeline. | 

### Return type

[**PaginatedPipelineSteps**](paginated_pipeline_steps.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineVariableForTeam**
> PipelineVariable GetPipelineVariableForTeam(ctx, username, variableUuid)


Retrieve a team level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable to retrieve. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineVariableForUser**
> PipelineVariable GetPipelineVariableForUser(ctx, username, variableUuid)


Retrieve a user level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable to retrieve. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineVariablesForTeam**
> PaginatedPipelineVariables GetPipelineVariablesForTeam(ctx, username)


Find account level variables.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 

### Return type

[**PaginatedPipelineVariables**](paginated_pipeline_variables.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelineVariablesForUser**
> PaginatedPipelineVariables GetPipelineVariablesForUser(ctx, username)


Find user level variables.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 

### Return type

[**PaginatedPipelineVariables**](paginated_pipeline_variables.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPipelinesForRepository**
> PaginatedPipelines GetPipelinesForRepository(ctx, username, repoSlug)


Find pipelines

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PaginatedPipelines**](paginated_pipelines.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineConfig**
> PipelinesConfig GetRepositoryPipelineConfig(ctx, username, repoSlug)


Retrieve the repository pipelines configuration.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PipelinesConfig**](pipelines_config.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineKnownHost**
> PipelineKnownHost GetRepositoryPipelineKnownHost(ctx, username, repoSlug, knownHostUuid)


Retrieve a repository level known host.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **knownHostUuid** | **string**| The UUID of the known host to retrieve. | 

### Return type

[**PipelineKnownHost**](pipeline_known_host.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineKnownHosts**
> PaginatedPipelineKnownHosts GetRepositoryPipelineKnownHosts(ctx, username, repoSlug)


Find repository level known hosts.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PaginatedPipelineKnownHosts**](paginated_pipeline_known_hosts.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineSchedule**
> PipelineSchedule GetRepositoryPipelineSchedule(ctx, username, repoSlug, scheduleUuid)


Retrieve a schedule by its UUID.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **scheduleUuid** | **string**| The uuid of the schedule. | 

### Return type

[**PipelineSchedule**](pipeline_schedule.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineScheduleExecutions**
> PaginatedPipelineScheduleExecutions GetRepositoryPipelineScheduleExecutions(ctx, username, repoSlug)


Retrieve the executions of a given schedule.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PaginatedPipelineScheduleExecutions**](paginated_pipeline_schedule_executions.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineSchedules**
> PaginatedPipelineSchedules GetRepositoryPipelineSchedules(ctx, username, repoSlug)


Retrieve the configured schedules for the given repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PaginatedPipelineSchedules**](paginated_pipeline_schedules.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineSshKeyPair**
> PipelineSshKeyPair GetRepositoryPipelineSshKeyPair(ctx, username, repoSlug)


Retrieve the repository SSH key pair excluding the SSH private key. The private key is a write only field and will never be exposed in the logs or the REST API.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PipelineSshKeyPair**](pipeline_ssh_key_pair.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineVariable**
> PipelineVariable GetRepositoryPipelineVariable(ctx, username, repoSlug, variableUuid)


Retrieve a repository level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **variableUuid** | **string**| The UUID of the variable to retrieve. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRepositoryPipelineVariables**
> PaginatedPipelineVariables GetRepositoryPipelineVariables(ctx, username, repoSlug)


Find repository level variables.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 

### Return type

[**PaginatedPipelineVariables**](paginated_pipeline_variables.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **StopPipeline**
> StopPipeline(ctx, username, repoSlug, pipelineUuid)


Signal the stop of a pipeline and all of its steps that not have completed yet.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **pipelineUuid** | **string**| The UUID of the pipeline. | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdatePipelineVariableForTeam**
> PipelineVariable UpdatePipelineVariableForTeam(ctx, username, variableUuid, body)


Update a team level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable. | 
  **body** | [**PipelineVariable**](PipelineVariable.md)| The updated variable. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdatePipelineVariableForUser**
> PipelineVariable UpdatePipelineVariableForUser(ctx, username, variableUuid, body)


Update a user level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **variableUuid** | **string**| The UUID of the variable. | 
  **body** | [**PipelineVariable**](PipelineVariable.md)| The updated variable. | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryBuildNumber**
> PipelineBuildNumber UpdateRepositoryBuildNumber(ctx, username, repoSlug, body)


Update the next build number that should be assigned to a pipeline. The next build number that will be configured has to be strictly higher than the current latest build number for this repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelineBuildNumber**](PipelineBuildNumber.md)| The build number to update. | 

### Return type

[**PipelineBuildNumber**](pipeline_build_number.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryPipelineConfig**
> PipelinesConfig UpdateRepositoryPipelineConfig(ctx, username, repoSlug, body)


Update the pipelines configuration for a repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelinesConfig**](PipelinesConfig.md)| The updated repository pipelines configuration. | 

### Return type

[**PipelinesConfig**](pipelines_config.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryPipelineKeyPair**
> PipelineSshKeyPair UpdateRepositoryPipelineKeyPair(ctx, username, repoSlug, body)


Create or update the repository SSH key pair. The private key will be set as a default SSH identity in your build container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **body** | [**PipelineSshKeyPair**](PipelineSshKeyPair.md)| The created or updated SSH key pair. | 

### Return type

[**PipelineSshKeyPair**](pipeline_ssh_key_pair.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryPipelineKnownHost**
> PipelineKnownHost UpdateRepositoryPipelineKnownHost(ctx, username, repoSlug, knownHostUuid, body)


Update a repository level known host.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **knownHostUuid** | **string**| The UUID of the known host to update. | 
  **body** | [**PipelineKnownHost**](PipelineKnownHost.md)| The updated known host. | 

### Return type

[**PipelineKnownHost**](pipeline_known_host.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryPipelineSchedule**
> PipelineSchedule UpdateRepositoryPipelineSchedule(ctx, username, repoSlug, scheduleUuid, body)


Update a schedule.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **scheduleUuid** | **string**| The uuid of the schedule. | 
  **body** | [**PipelineSchedule**](PipelineSchedule.md)| The schedule to update. | 

### Return type

[**PipelineSchedule**](pipeline_schedule.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRepositoryPipelineVariable**
> PipelineVariable UpdateRepositoryPipelineVariable(ctx, username, repoSlug, variableUuid, body)


Update a repository level variable.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **username** | **string**| The account. | 
  **repoSlug** | **string**| The repository. | 
  **variableUuid** | **string**| The UUID of the variable to update. | 
  **body** | [**PipelineVariable**](PipelineVariable.md)| The updated variable | 

### Return type

[**PipelineVariable**](pipeline_variable.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

