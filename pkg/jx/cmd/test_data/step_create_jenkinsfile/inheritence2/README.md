This test case shows how the [pipeline.yaml](pipeline.yaml) can replace or add init steps on top of the [base-pipeline.yaml](base-pipeline.yaml) pipeline via inheritance.

* to add steps before a base pipeline use `initSteps` like [this example](pipeline.yaml#L6-L9)
* to add steps after a base pipeline use `steps` like [this example](pipeline.yaml#L10-L13)
* to replace the steps from the base use `steps` and specify `replace: true` like [this example](pipeline.yaml#L15-L19)
