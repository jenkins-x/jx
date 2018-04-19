# \ServicesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteService**](ServicesApi.md#DeleteService) | **Delete** /system/services/{servicename}/{hostid} | Delete the service config
[**GetServicesByName**](ServicesApi.md#GetServicesByName) | **Get** /system/services/{servicename} | Get a service configuration and state
[**GetServicesByNameAndHost**](ServicesApi.md#GetServicesByNameAndHost) | **Get** /system/services/{servicename}/{hostid} | Get service config for a specific host
[**ListServices**](ServicesApi.md#ListServices) | **Get** /system/services | List system services


# **DeleteService**
> DeleteService(ctx, servicename, hostid)
Delete the service config

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **servicename** | **string**|  | 
  **hostid** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetServicesByName**
> ServiceList GetServicesByName(ctx, servicename)
Get a service configuration and state

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **servicename** | **string**|  | 

### Return type

[**ServiceList**](ServiceList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetServicesByNameAndHost**
> ServiceList GetServicesByNameAndHost(ctx, servicename, hostid)
Get service config for a specific host

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **servicename** | **string**|  | 
  **hostid** | **string**|  | 

### Return type

[**ServiceList**](ServiceList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListServices**
> ServiceList ListServices(ctx, )
List system services

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**ServiceList**](ServiceList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

