# Experiments and feature flags

This page contains a list of experiments that are being worked on and how to enable them.  This list will be maintained and
it will likely change how features are enabled as they mature past an experiment. 

## Helmfile

We are experimenting with [Helmfile](https://github.com/roboll/helmfile) to see if we can make the `jx boot` implementation
a bit more modular and leverage some of the extra fetures the OSS project has.  To enable this feature set the `JX_HELMFILE`
environment variable:

```bash
export JX_HELMFILE=true
```