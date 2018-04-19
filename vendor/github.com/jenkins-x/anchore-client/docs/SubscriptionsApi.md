# \SubscriptionsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddSubscription**](SubscriptionsApi.md#AddSubscription) | **Post** /subscriptions | Add a subscription of a specific type
[**DeleteSubscription**](SubscriptionsApi.md#DeleteSubscription) | **Delete** /subscriptions/{subscriptionId} | Delete subscriptions of a specific type
[**GetSubscription**](SubscriptionsApi.md#GetSubscription) | **Get** /subscriptions/{subscriptionId} | Get a specific subscription set
[**ListSubscriptions**](SubscriptionsApi.md#ListSubscriptions) | **Get** /subscriptions | List all subscriptions
[**UpdateSubscription**](SubscriptionsApi.md#UpdateSubscription) | **Put** /subscriptions/{subscriptionId} | Update an existing and specific subscription


# **AddSubscription**
> Subscription AddSubscription(ctx, subscription)
Add a subscription of a specific type

Create a new subscription to watch a tag and get notifications of changes

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **subscription** | [**SubscriptionRequest**](SubscriptionRequest.md)|  | 

### Return type

[**Subscription**](Subscription.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteSubscription**
> DeleteSubscription(ctx, subscriptionId)
Delete subscriptions of a specific type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **subscriptionId** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSubscription**
> SubscriptionList GetSubscription(ctx, subscriptionId)
Get a specific subscription set

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **subscriptionId** | **string**|  | 

### Return type

[**SubscriptionList**](SubscriptionList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListSubscriptions**
> SubscriptionList ListSubscriptions(ctx, )
List all subscriptions

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**SubscriptionList**](SubscriptionList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateSubscription**
> Subscription UpdateSubscription(ctx, subscriptionId, subscription)
Update an existing and specific subscription

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **subscriptionId** | **string**|  | 
  **subscription** | [**SubscriptionUpdate**](SubscriptionUpdate.md)|  | 

### Return type

[**Subscription**](Subscription.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

