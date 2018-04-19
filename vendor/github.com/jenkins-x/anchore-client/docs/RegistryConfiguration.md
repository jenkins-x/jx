# RegistryConfiguration

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | [**time.Time**](time.Time.md) |  | [optional] [default to null]
**LastUpated** | [**time.Time**](time.Time.md) |  | [optional] [default to null]
**Registry** | **string** | hostname:port string for accessing the registry, as would be used in a docker pull operation | [optional] [default to null]
**RegistryType** | **string** | Type of registry | [optional] [default to null]
**RegistryUser** | **string** | Username portion of credential to use for this registry | [optional] [default to null]
**RegistryVerify** | **bool** | Use TLS/SSL verification for the registry URL | [optional] [default to null]
**UserId** | **string** | Engine user that owns this registry entry | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


