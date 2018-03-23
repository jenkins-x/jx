# Pipeline

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**BuildNumber** | **int32** | The build number of the pipeline. | [optional] [default to null]
**Target** | [***PipelineTarget**](pipeline_target.md) | The target that the pipeline built. | [optional] [default to null]
**Repository** | [***Repository**](repository.md) |  | [optional] [default to null]
**Creator** | [***Account**](account.md) | The Bitbucket account that was used to create the pipeline. | [optional] [default to null]
**CreatedOn** | [**time.Time**](time.Time.md) | The timestamp when the pipeline was created. | [optional] [default to null]
**State** | [***PipelineState**](pipeline_state.md) |  | [optional] [default to null]
**Trigger** | [***PipelineTrigger**](pipeline_trigger.md) | The trigger used for the pipeline. | [optional] [default to null]
**BuildSecondsUsed** | **int32** | The number of build seconds used by this pipeline. | [optional] [default to null]
**CompletedOn** | [**time.Time**](time.Time.md) | The timestamp when the Pipeline was completed. This is not set if the pipeline is still in progress. | [optional] [default to null]
**Uuid** | **string** | The UUID identifying the pipeline. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


