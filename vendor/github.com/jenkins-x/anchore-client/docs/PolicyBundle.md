# PolicyBundle

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**BlacklistedImages** | [**[]ImageSelectionRule**](ImageSelectionRule.md) | List of mapping rules that define which images should always result in a STOP/FAIL policy result regardless of policy content or presence in whitelisted_images | [optional] [default to null]
**Comment** | **string** | Description of the bundle, human readable | [optional] [default to null]
**Id** | **string** | Id of the bundle | [default to null]
**Mappings** | [**[]MappingRule**](MappingRule.md) | Mapping rules for defining which policy and whitelist(s) to apply to an image based on a match of the image tag or id. Evaluated in order. | [default to null]
**Name** | **string** | Human readable name for the bundle | [optional] [default to null]
**Policies** | [**[]Policy**](Policy.md) | Policies which define the go/stop/warn status of an image using rule matches on image properties | [default to null]
**Version** | **string** | Version id for this bundle format | [default to null]
**WhitelistedImages** | [**[]ImageSelectionRule**](ImageSelectionRule.md) | List of mapping rules that define which images should always be passed (unless also on the blacklist), regardless of policy result. | [optional] [default to null]
**Whitelists** | [**[]Whitelist**](Whitelist.md) | Whitelists which define which policy matches to disregard explicitly in the final policy decision | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


