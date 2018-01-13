# JX 

JX is a CLI tool for working with [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/)

## Installing

On a Mac you can use brew:

    brew tap jenkins-x/jx
    brew install jx
    
Or download the binary `jx` and add it to your `$PATH`

## Quickstart

To find out the available commands type:

    jx

Or to get help on a specific command, say, 'create' then type:

    jx help create
     
If you don't yet have a kubernetes cluster then try:

    jx create cluster
 
Otherwise you can install Jenkins X in your current kubernetes cluster via:

    jx init
    
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
                                            
