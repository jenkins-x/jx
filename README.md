# JX 

JX is a CLI tool for working with [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/)

## Installing

On a Mac you can use brew:

    brew install Jenkins-x/jx/jx 
    
Or download the binary `jx` and add it to your `$PATH`

## Getting Help

To find out the available commands type:

    jx

Or to get help on a specific command, say, 'create' then type:

    jx help create

## Getting Started

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
    

## Viewing Apps and Environments

To view environments for a team

    jx get env
    
To view the application versions across environments

    jx get version
            
## Switching Environments

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

On a Mac to enable bash completion try:

    jx completion bash > ~/.jx/bash
    source ~/.jx/bash   
    
Or try:

    source <(jx completion bash)

For more help try:

    jx help completion bash
     
                                            
