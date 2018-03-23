# Pullrequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**Links** | [***PullrequestLinks**](pullrequest_links.md) |  | [optional] [default to null]
**Id** | **int32** | The pull request&#39;s unique ID. Note that pull request IDs are only unique within their associated repository. | [optional] [default to null]
**Title** | **string** | Title of the pull request. | [optional] [default to null]
**Summary** | [***PullrequestSummary**](pullrequest_summary.md) |  | [optional] [default to null]
**State** | **string** | The pull request&#39;s current status. | [optional] [default to null]
**Author** | [***Account**](account.md) |  | [optional] [default to null]
**Source** | [***PullrequestEndpoint**](pullrequest_endpoint.md) |  | [optional] [default to null]
**Destination** | [***PullrequestEndpoint**](pullrequest_endpoint.md) |  | [optional] [default to null]
**MergeCommit** | [***PullrequestMergeCommit**](pullrequest_merge_commit.md) |  | [optional] [default to null]
**CommentCount** | **int32** | The number of comments for a specific pull request. | [optional] [default to null]
**TaskCount** | **int32** | The number of open tasks for a specific pull request. | [optional] [default to null]
**CloseSourceBranch** | **bool** | A boolean flag indicating if merging the pull request closes the source branch. | [optional] [default to null]
**ClosedBy** | [***Account**](account.md) |  | [optional] [default to null]
**Reason** | **string** | Explains why a pull request was declined. This field is only applicable to pull requests in rejected state. | [optional] [default to null]
**CreatedOn** | [**time.Time**](time.Time.md) | The ISO8601 timestamp the request was created. | [optional] [default to null]
**UpdatedOn** | [**time.Time**](time.Time.md) | The ISO8601 timestamp the request was last updated. | [optional] [default to null]
**Reviewers** | [**[]Account**](account.md) | The list of users that were added as reviewers on this pull request when it was created. For performance reasons, the API only includes this list on a pull request&#39;s &#x60;self&#x60; URL. | [optional] [default to null]
**Participants** | [**[]Participant**](participant.md) |         The list of users that are collaborating on this pull request.         Collaborators are user that:          * are added to the pull request as a reviewer (part of the reviewers           list)         * are not explicit reviewers, but have commented on the pull request         * are not explicit reviewers, but have approved the pull request          Each user is wrapped in an object that indicates the user&#39;s role and         whether they have approved the pull request. For performance reasons,         the API only returns this list when an API requests a pull request by         id.          | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


