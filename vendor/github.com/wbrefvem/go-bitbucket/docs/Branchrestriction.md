# Branchrestriction

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**Links** | [***MilestoneLinks**](milestone_links.md) |  | [optional] [default to null]
**Id** | **int32** | The branch restriction status&#39; id. | [optional] [default to null]
**Kind** | **string** | The type of restriction that is being applied | [optional] [default to null]
**Users** | [**[]Account**](account.md) |  | [optional] [default to null]
**Groups** | [**[]Group**](group.md) |  | [optional] [default to null]
**Value** | **int32** | Value with kind-specific semantics: \&quot;require_approvals_to_merge\&quot; uses it to require a minimum number of approvals on a PR; \&quot;require_passing_builds_to_merge\&quot; uses it to require a minimum number of passing builds. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


