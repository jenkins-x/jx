# JX 

JX is a command line tool for installing and using [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/)

## Installing

On a Mac you can use brew:

    brew tap jenkins-x/jx
    brew install jx 
    
Or [download the binary](https://github.com/jenkins-x/jx/releases) for `jx` and add it to your `$PATH`

Or you can try [build it yourself](https://github.com/jenkins-x/jx/blob/master/docs/contributing/hacking.md). Though if build it yourself please be careful to remove any older `jx` binary so your local build is found first on the `$PATH` :)

## Getting Help

To find out the available commands type:

    jx

Or to get help on a specific command, say, `create` then type:

    jx help create

## Getting Started

The quickest way to get started is to use the `jx create cluster` command - this will create the cluster, install client side dependencies and provision the Jenkins X platform.

If you don't have access to a kubernetes cluster then using [minikube](https://github.com/kubernetes/minikube#minikube) is a great way to kick the tires locally on your laptop. 

    jx create cluster minikube

If that does not work first time for you then please [let us know](https://github.com/jenkins-x/jx/issues/new). A work around is to try [install minikube yourself](https://github.com/kubernetes/minikube#installation) and [start it up](https://github.com/kubernetes/minikube#quickstart) then use `jx install` as described below.

### Using an existing kubernetes cluster

If you already have a kubernetes cluster setup then try:

    jx install
        
## Importing or Creating apps

To import an application from the current directory:

    jx import
    
Or to create a new Spring Boot application from scratch:

    jx create spring
    
e.g. to create a new WebMVC and Spring Boot Actuator microservice try this:

    jx create spring -d web -d actuator
        
If you have a Maven Archetype you would like to create then use:

    jx create archetype
    

## Viewing Apps and Environments

To view environments for a team

    jx get env
    
To view the application versions across environments

    jx get version
            
## Manual promotions

Typically we setup Environments to use _automatic_ promotion so that the CI / CD pipelines will automatically promote versions through the available Environments using the CI / CD Pipeline.

However if you wish to manually promote a version to an environment you can use the following command:

    jx promote myapp -e prod 
    
Or if you wish to use a custom namespace    

    jx promote myapp -n my-dummy-namespace
 
## Switching Environments

The `jx` CLI tool uses the same kubernetes cluster and namespace context as `kubectl`. 

You can switch Environments via:

    jx env
    
Or change it via 

    jx env staging
    jx env prod
    
To display the current environment without trying to change it:

    jx env -b

To view all the environments type:

    jx get env
    
You can create or edit environments too

    jx create env
    
    jx edit env staging
    
You can switch namespaces in the same way via

    jx ns

or

    jx ns awesome-staging    
    
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
           
### Setting your prompt

You can use the `jx prompt` to configure your CLI prompt to display the current team and environment you are working within           
                                            
		# Enable the prompt for bash
		PS1="[\u@\h \W \$(jx prompt)]\$ "

		# Enable the prompt for zsh
		PROMPT='$(jx prompt)'$PROMPT

### Uninstall Jenkins x

To remove the Jenkins X platfrom from a namespace on your kubernetes cluster:

    jx uninstall
	
## Contributing

If you're looking to build from source or get started hacking on jx, please see the
[hacking guide][hacking] for more information.


[hacking]: docs/contributing/hacking.md
