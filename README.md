# JX 

JX is a CLI tool to help work with [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/)

## Quickstart

To find out the available commands type:

    jx help
    
Once you have `jx` downloaded and on your `$PATH` you will want to create a kubernetes cluster:

    jx create cluster
    
## Importing or Creating apps

To import a local application try:

    jx import
    
Or to create a new Spring Boot application from scratch

    jx create spring
    
If you have a Maven Archetype you would like to create then use:

    jx create archetype
    
    
### Changing Environments

The `jx` CLI tool uses the same kubernetes cluster and namespace context as `kubectl`. 

You can view your current Environment via:

    jx env
    
Or change it via 

    jx env staging
    jx env prod
    
To pick which environment you with to change to try:

    jx env -s

To view environments type:

    jx get env
    
You can create or edit environments too

    jx create env
    
    jx edit env staging
    
You can change namespaces in the same way via

    jx ns
    
### Tail logs

To tail the logs of an app type

    jx logs
    
Which by default tails the logs of the newest pod for an app.

### Opening Consoles

To open the Jenkins console try:

    jx console
    
Or to open other consoles

    jx open foo
    
If you do not know the name

    jx open
    

### Bash completion

To enable bash completion see

    jx help completion bash
                                            
