# \PoliciesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddPolicy**](PoliciesApi.md#AddPolicy) | **Post** /policies | Add a new policy
[**DeletePolicy**](PoliciesApi.md#DeletePolicy) | **Delete** /policies/{policyId} | Delete policy
[**GetPolicy**](PoliciesApi.md#GetPolicy) | **Get** /policies/{policyId} | Get specific policy
[**ListPolicies**](PoliciesApi.md#ListPolicies) | **Get** /policies | List policies
[**UpdatePolicy**](PoliciesApi.md#UpdatePolicy) | **Put** /policies/{policyId} | Update policy


# **AddPolicy**
> PolicyBundleRecord AddPolicy(ctx, bundle)
Add a new policy

Adds a new policy bundle to the system

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **bundle** | [**PolicyBundle**](PolicyBundle.md)|  | 

### Return type

[**PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeletePolicy**
> DeletePolicy(ctx, policyId)
Delete policy

Delete the specified policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **policyId** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPolicy**
> PolicyBundleRecord GetPolicy(ctx, policyId, optional)
Get specific policy

Get the policy bundle content

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **policyId** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **policyId** | **string**|  | 
 **detail** | **bool**| Include policy bundle detail in the form of the full bundle content for each entry | 

### Return type

[**PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListPolicies**
> PolicyBundleList ListPolicies(ctx, optional)
List policies

List all saved policy bundles

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **detail** | **bool**| Include policy bundle detail in the form of the full bundle content for each entry | 

### Return type

[**PolicyBundleList**](PolicyBundleList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdatePolicy**
> PolicyBundleRecord UpdatePolicy(ctx, bundle, policyId, optional)
Update policy

Update/replace and existing policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **bundle** | [**PolicyBundleRecord**](PolicyBundleRecord.md)|  | 
  **policyId** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **bundle** | [**PolicyBundleRecord**](PolicyBundleRecord.md)|  | 
 **policyId** | **string**|  | 
 **active** | **bool**| Mark policy as active | 

### Return type

[**PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

