# \RegistriesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateRegistry**](RegistriesApi.md#CreateRegistry) | **Post** /registries | Add a new registry
[**DeleteRegistry**](RegistriesApi.md#DeleteRegistry) | **Delete** /registries/{registry} | Delete a registry configuration
[**GetRegistry**](RegistriesApi.md#GetRegistry) | **Get** /registries/{registry} | Get a specific registry configuration
[**ListRegistries**](RegistriesApi.md#ListRegistries) | **Get** /registries | List configured registries
[**UpdateRegistry**](RegistriesApi.md#UpdateRegistry) | **Put** /registries/{registry} | Update/replace a registry configuration


# **CreateRegistry**
> RegistryConfiguration CreateRegistry(ctx, registrydata)
Add a new registry

Adds a new registry to the system

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **registrydata** | [**RegistryConfiguration**](RegistryConfiguration.md)|  | 

### Return type

[**RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRegistry**
> DeleteRegistry(ctx, registry)
Delete a registry configuration

Delete a registry configuration record from the system. Does not remove any images.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **registry** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRegistry**
> RegistryConfiguration GetRegistry(ctx, registry)
Get a specific registry configuration

Get information on a specific registry

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **registry** | **string**|  | 

### Return type

[**RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListRegistries**
> RegistryConfigurationList ListRegistries(ctx, )
List configured registries

List all configured registries the system can/will watch

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**RegistryConfigurationList**](RegistryConfigurationList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRegistry**
> RegistryConfiguration UpdateRegistry(ctx, registry, registrydata)
Update/replace a registry configuration

Replaces an existing registry record with the given record

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **registry** | **string**|  | 
  **registrydata** | [**RegistryConfiguration**](RegistryConfiguration.md)|  | 

### Return type

[**RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

