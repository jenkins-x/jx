This test case shows how the [pipeline.yaml](pipeline.yaml) layers on the kubernetes workloads capabilities on top of the [base-pipeline.yaml](base-pipeline.yaml) pipeline via inheritance.

By default steps are appended to the base pipeline at each lifecycle section (`setup`, `preBuild`, `build`, `postBuild`, `promote` etc)
