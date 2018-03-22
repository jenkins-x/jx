# Participant

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**User** | [***User**](user.md) |  | [optional] [default to null]
**Role** | **string** |  | [optional] [default to null]
**Approved** | **bool** |  | [optional] [default to null]
**ParticipatedOn** | [**time.Time**](time.Time.md) | The ISO8601 timestamp of the participant&#39;s action. For approvers, this is the time of their approval. For commenters and pull request reviewers who are not approvers, this is the time they last commented, or null if they have not commented. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


