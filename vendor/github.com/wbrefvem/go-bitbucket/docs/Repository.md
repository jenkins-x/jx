# Repository

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**Links** | [***RepositoryLinks**](repository_links.md) |  | [optional] [default to null]
**Uuid** | **string** | The repository&#39;s immutable id. This can be used as a substitute for the slug segment in URLs. Doing this guarantees your URLs will survive renaming of the repository by its owner, or even transfer of the repository to a different user. | [optional] [default to null]
**FullName** | **string** | The concatenation of the repository owner&#39;s username and the slugified name, e.g. \&quot;evzijst/interruptingcow\&quot;. This is the same string used in Bitbucket URLs. | [optional] [default to null]
**IsPrivate** | **bool** |  | [optional] [default to null]
**Parent** | [***Repository**](repository.md) |  | [optional] [default to null]
**Scm** | **string** |  | [optional] [default to null]
**Owner** | [***Account**](account.md) |  | [optional] [default to null]
**Name** | **string** |  | [optional] [default to null]
**Description** | **string** |  | [optional] [default to null]
**CreatedOn** | [**time.Time**](time.Time.md) |  | [optional] [default to null]
**UpdatedOn** | [**time.Time**](time.Time.md) |  | [optional] [default to null]
**Size** | **int32** |  | [optional] [default to null]
**Language** | **string** |  | [optional] [default to null]
**HasIssues** | **bool** |  | [optional] [default to null]
**HasWiki** | **bool** |  | [optional] [default to null]
**ForkPolicy** | **string** |  Controls the rules for forking this repository.  * **allow_forks**: unrestricted forking * **no_public_forks**: restrict forking to private forks (forks cannot   be made public later) * **no_forks**: deny all forking  | [optional] [default to null]
**Project** | [***Project**](project.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


