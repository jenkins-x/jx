# \AddonApi

All URIs are relative to *https://api.bitbucket.org/2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddonDelete**](AddonApi.md#AddonDelete) | **Delete** /addon | 
[**AddonLinkersGet**](AddonApi.md#AddonLinkersGet) | **Get** /addon/linkers | 
[**AddonLinkersLinkerKeyGet**](AddonApi.md#AddonLinkersLinkerKeyGet) | **Get** /addon/linkers/{linker_key} | 
[**AddonLinkersLinkerKeyValuesDelete**](AddonApi.md#AddonLinkersLinkerKeyValuesDelete) | **Delete** /addon/linkers/{linker_key}/values | 
[**AddonLinkersLinkerKeyValuesDelete_0**](AddonApi.md#AddonLinkersLinkerKeyValuesDelete_0) | **Delete** /addon/linkers/{linker_key}/values/ | 
[**AddonLinkersLinkerKeyValuesGet**](AddonApi.md#AddonLinkersLinkerKeyValuesGet) | **Get** /addon/linkers/{linker_key}/values | 
[**AddonLinkersLinkerKeyValuesGet_0**](AddonApi.md#AddonLinkersLinkerKeyValuesGet_0) | **Get** /addon/linkers/{linker_key}/values/ | 
[**AddonLinkersLinkerKeyValuesPost**](AddonApi.md#AddonLinkersLinkerKeyValuesPost) | **Post** /addon/linkers/{linker_key}/values | 
[**AddonLinkersLinkerKeyValuesPut**](AddonApi.md#AddonLinkersLinkerKeyValuesPut) | **Put** /addon/linkers/{linker_key}/values | 
[**AddonPut**](AddonApi.md#AddonPut) | **Put** /addon | 


# **AddonDelete**
> ModelError AddonDelete(ctx, )




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

# **AddonLinkersGet**
> ModelError AddonLinkersGet(ctx, )




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

# **AddonLinkersLinkerKeyGet**
> ModelError AddonLinkersLinkerKeyGet(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesDelete**
> ModelError AddonLinkersLinkerKeyValuesDelete(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesDelete_0**
> ModelError AddonLinkersLinkerKeyValuesDelete_0(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesGet**
> ModelError AddonLinkersLinkerKeyValuesGet(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesGet_0**
> ModelError AddonLinkersLinkerKeyValuesGet_0(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesPost**
> ModelError AddonLinkersLinkerKeyValuesPost(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonLinkersLinkerKeyValuesPut**
> ModelError AddonLinkersLinkerKeyValuesPut(ctx, linkerKey)




### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **linkerKey** | **string**|  | 

### Return type

[**ModelError**](error.md)

### Authorization

[api_key](../README.md#api_key), [basic](../README.md#basic), [oauth2](../README.md#oauth2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddonPut**
> ModelError AddonPut(ctx, )




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

