# \ImagesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddImage**](ImagesApi.md#AddImage) | **Post** /images | Submit a new image for analysis by the engine
[**DeleteImage**](ImagesApi.md#DeleteImage) | **Delete** /images/{imageDigest} | Delete an image analysis
[**DeleteImageByImageId**](ImagesApi.md#DeleteImageByImageId) | **Delete** /images/by_id/{imageId} | Delete image by docker imageId
[**GetImage**](ImagesApi.md#GetImage) | **Get** /images/{imageDigest} | Get image metadata
[**GetImageByImageId**](ImagesApi.md#GetImageByImageId) | **Get** /images/by_id/{imageId} | Lookup image by docker imageId
[**GetImageContentByType**](ImagesApi.md#GetImageContentByType) | **Get** /images/{imageDigest}/content/{ctype} | Get the content of an image by type
[**GetImageContentByTypeImageId**](ImagesApi.md#GetImageContentByTypeImageId) | **Get** /images/by_id/{imageId}/content/{ctype} | Get the content of an image by type
[**GetImagePolicyCheck**](ImagesApi.md#GetImagePolicyCheck) | **Get** /images/{imageDigest}/check | Check policy evaluation status for image
[**GetImagePolicyCheckByImageId**](ImagesApi.md#GetImagePolicyCheckByImageId) | **Get** /images/by_id/{imageId}/check | Check policy evaluation status for image
[**GetImageVulnerabilitiesByType**](ImagesApi.md#GetImageVulnerabilitiesByType) | **Get** /images/{imageDigest}/vuln/{vtype} | Get vulnerabilities by type
[**GetImageVulnerabilitiesByTypeImageId**](ImagesApi.md#GetImageVulnerabilitiesByTypeImageId) | **Get** /images/by_id/{imageId}/vuln/{vtype} | Get vulnerabilities by type
[**GetImageVulnerabilityTypes**](ImagesApi.md#GetImageVulnerabilityTypes) | **Get** /images/{imageDigest}/vuln | Get vulnerability types
[**GetImageVulnerabilityTypesByImageId**](ImagesApi.md#GetImageVulnerabilityTypesByImageId) | **Get** /images/by_id/{imageId}/vuln | Get vulnerability types
[**ImportImage**](ImagesApi.md#ImportImage) | **Post** /imageimport | Import and image analysis directly
[**ListImageContent**](ImagesApi.md#ListImageContent) | **Get** /images/{imageDigest}/content | List image content types
[**ListImageContentByImageid**](ImagesApi.md#ListImageContentByImageid) | **Get** /images/by_id/{imageId}/content | List image content types
[**ListImages**](ImagesApi.md#ListImages) | **Get** /images | List all visible images


# **AddImage**
> []AnchoreImage AddImage(ctx, image, optional)
Submit a new image for analysis by the engine

Creates a new analysis task that is executed asynchronously

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **image** | [**ImageAnalysisRequest**](ImageAnalysisRequest.md)|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **image** | [**ImageAnalysisRequest**](ImageAnalysisRequest.md)|  | 
 **force** | **bool**| Override any existing entry in the system | 
 **autosubscribe** | **bool**| Instruct engine to automatically begin watching the added tag for updates from registry | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteImage**
> DeleteImage(ctx, imageDigest)
Delete an image analysis

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteImageByImageId**
> DeleteImageByImageId(ctx, imageId, optional)
Delete image by docker imageId

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **imageId** | **string**|  | 
 **force** | **bool**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImage**
> []AnchoreImage GetImage(ctx, imageDigest)
Get image metadata

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageByImageId**
> []AnchoreImage GetImageByImageId(ctx, imageId)
Lookup image by docker imageId

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageContentByType**
> ContentResponse GetImageContentByType(ctx, imageDigest, ctype)
Get the content of an image by type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 
  **ctype** | **string**|  | 

### Return type

[**ContentResponse**](ContentResponse.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageContentByTypeImageId**
> ContentResponse GetImageContentByTypeImageId(ctx, imageId, ctype)
Get the content of an image by type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 
  **ctype** | **string**|  | 

### Return type

[**ContentResponse**](ContentResponse.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

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

# **GetImageVulnerabilitiesByType**
> VulnerabilityList GetImageVulnerabilitiesByType(ctx, imageDigest, vtype)
Get vulnerabilities by type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 
  **vtype** | **string**|  | 

### Return type

[**VulnerabilityList**](VulnerabilityList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageVulnerabilitiesByTypeImageId**
> VulnerabilityList GetImageVulnerabilitiesByTypeImageId(ctx, imageId, vtype)
Get vulnerabilities by type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 
  **vtype** | **string**|  | 

### Return type

[**VulnerabilityList**](VulnerabilityList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageVulnerabilityTypes**
> []string GetImageVulnerabilityTypes(ctx, imageDigest)
Get vulnerability types

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 

### Return type

**[]string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetImageVulnerabilityTypesByImageId**
> []string GetImageVulnerabilityTypesByImageId(ctx, imageId)
Get vulnerability types

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 

### Return type

**[]string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ImportImage**
> []AnchoreImage ImportImage(ctx, analysisReport)
Import and image analysis directly

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **analysisReport** | [**ImageAnalysisReport**](ImageAnalysisReport.md)|  | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListImageContent**
> []string ListImageContent(ctx, imageDigest)
List image content types

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageDigest** | **string**|  | 

### Return type

**[]string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListImageContentByImageid**
> []string ListImageContentByImageid(ctx, imageId)
List image content types

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **imageId** | **string**|  | 

### Return type

**[]string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListImages**
> []AnchoreImage ListImages(ctx, optional)
List all visible images

List all images visible to the user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
 **optional** | **map[string]interface{}** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a map[string]interface{}.

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **history** | **bool**| Include image history in the response | 
 **imageToGet** | [**ImageFilter**](ImageFilter.md)|  | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

