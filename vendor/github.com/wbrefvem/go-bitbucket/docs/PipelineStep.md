# PipelineStep

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** |  | [default to null]
**CompletedOn** | [**time.Time**](time.Time.md) | The timestamp when the step execution was completed. This is not set if the step is still in progress. | [optional] [default to null]
**LogByteCount** | **int32** | The amount of bytes of the log file that is available. | [optional] [default to null]
**Image** | [***PipelineImage**](pipeline_image.md) | The Docker image used as the build container for the step. | [optional] [default to null]
**StartedOn** | [**time.Time**](time.Time.md) | The timestamp when the step execution was started. This is not set when the step hasn&#39;t executed yet. | [optional] [default to null]
**ScriptCommands** | [**[]PipelineCommand**](pipeline_command.md) | The list of build commands. These commands are executed in the build container. | [optional] [default to null]
**State** | [***PipelineStepState**](pipeline_step_state.md) | The current state of the step | [optional] [default to null]
**SetupCommands** | [**[]PipelineCommand**](pipeline_command.md) | The list of commands that are executed as part of the setup phase of the build. These commands are executed outside the build container. | [optional] [default to null]
**Uuid** | **string** | The UUID identifying the step. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


