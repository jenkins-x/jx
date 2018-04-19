# \PolicyEvaluationApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetImagePolicyCheck**](PolicyEvaluationApi.md#GetImagePolicyCheck) | **Get** /images/{imageDigest}/check | Check policy evaluation status for image
[**GetImagePolicyCheckByImageId**](PolicyEvaluationApi.md#GetImagePolicyCheckByImageId) | **Get** /images/by_id/{imageId}/check | Check policy evaluation status for image


# **GetImagePolicyCheck**
> PolicyEvaluation GetImagePolicyCheck(ctx, imageDigest, tag, optional)
Check policy evaluation status for image

Get the policy evaluation for the given image

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 
  **tag** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **imageDigest** | **string**|  | 
 **tag** | **string**|  | 
 **policyId** | **string**|  | 
 **detail** | **bool**|  | 
 **history** | **bool**|  | 

### Return type

[**PolicyEvaluation**](PolicyEvaluation.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImagePolicyCheckByImageId**
> PolicyEvaluation GetImagePolicyCheckByImageId(ctx, imageId, tag, optional)
Check policy evaluation status for image

Get the policy evaluation for the given image

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 
  **tag** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **imageId** | **string**|  | 
 **tag** | **string**|  | 
 **policyId** | **string**|  | 
 **detail** | **bool**|  | 
 **history** | **bool**|  | 

### Return type

[**PolicyEvaluation**](PolicyEvaluation.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

