# Commitstatus

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**Links** | [***CommitstatusLinks**](commitstatus_links.md) |  | [optional] [default to null]
**Uuid** | **string** | The commit status&#39; id. | [optional] [default to null]
**Key** | **string** | An identifier for the status that&#39;s unique to         its type (current \&quot;build\&quot; is the only supported type) and the vendor,         e.g. BB-DEPLOY | [optional] [default to null]
**Refname** | **string** |  The name of the ref that pointed to this commit at the time the status object was created. Note that this the ref may since have moved off of the commit. This optional field can be useful for build systems whose build triggers and configuration are branch-dependent (e.g. a Pipeline build). It is legitimate for this field to not be set, or even apply (e.g. a static linting job). | [optional] [default to null]
**Url** | **string** | A URL linking back to the vendor or build system, for providing more information about whatever process produced this status. Accepts context variables &#x60;repository&#x60; and &#x60;commit&#x60; that Bitbucket will evaluate at runtime whenever at runtime. For example, one could use https://foo.com/builds/{repository.full_name} which Bitbucket will turn into https://foo.com/builds/foo/bar at render time. | [optional] [default to null]
**State** | **string** | Provides some indication of the status of this commit | [optional] [default to null]
**Name** | **string** | An identifier for the build itself, e.g. BB-DEPLOY-1 | [optional] [default to null]
**Description** | **string** | A description of the build (e.g. \&quot;Unit tests in Bamboo\&quot;) | [optional] [default to null]
**CreatedOn** | [**time.Time**](time.Time.md) |  | [optional] [default to null]
**UpdatedOn** | [**time.Time**](time.Time.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


