# \CatalogApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddRepository**](CatalogApi.md#AddRepository) | **Post** /repositories | Add repository to watch
[**GetSystemPruneCandidates**](CatalogApi.md#GetSystemPruneCandidates) | **Get** /system/prune/{resourcetype} | Get list of candidates for pruning
[**GetSystemPruneResourcetypes**](CatalogApi.md#GetSystemPruneResourcetypes) | **Get** /system/prune | Get list of resources that can be pruned
[**PostSystemPruneCandidates**](CatalogApi.md#PostSystemPruneCandidates) | **Post** /system/prune/{resourcetype} | Perform pruning on input resource name


# **AddRepository**
> RepositoryTagList AddRepository(ctx, repository, optional)
Add repository to watch



### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **repository** | **string**| full repository to add e.g. docker.io/library/alpine | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **repository** | **string**| full repository to add e.g. docker.io/library/alpine | 
 **autosubscribe** | **bool**| flag to enable/disable auto tag_update activation when new images from a repo are added | 
 **lookuptag** | **string**| use specified existing tag to perform repo scan (default is &#39;latest&#39;) | 

### Return type

[**RepositoryTagList**](RepositoryTagList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSystemPruneCandidates**
> PruneCandidateList GetSystemPruneCandidates(ctx, resourcetype, optional)
Get list of candidates for pruning



### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **resourcetype** | **string**| resource type | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **resourcetype** | **string**| resource type | 
 **dangling** | **bool**| filter only disconnected resources | 
 **olderthan** | **int32**| filter only resources older than specified number of seconds | 

### Return type

[**PruneCandidateList**](PruneCandidateList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSystemPruneResourcetypes**
> []string GetSystemPruneResourcetypes(ctx, )
Get list of resources that can be pruned



### Required Parameters
This endpoint does not need any parameter.

### Return type

**[]string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostSystemPruneCandidates**
> PruneCandidateList PostSystemPruneCandidates(ctx, resourcetype, bodycontent)
Perform pruning on input resource name



### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **resourcetype** | **string**| resource type | 
  **bodycontent** | [**PruneCandidate**](PruneCandidate.md)| resource objects to prune | 

### Return type

[**PruneCandidateList**](PruneCandidateList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

