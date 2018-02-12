# JX 

JX is a command line tool for installing and using [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/)

<a href="https://jenkins-x.github.io/jenkins-x-website/">
  <img src="https://jenkins-x.github.io/jenkins-x-website/img/profile.png" alt="Jenkins X icon" width="100" height="97"/>
</a>

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

You can also browse the [jx command reference documentation](https://github.com/jenkins-x/documentation/blob/master/_docs/reference/jx.md)

## Getting Started

### Installing Kubernetes with Minikube on Mac

The quickest way to get started is to install Minikube using Homebrew:

    brew cask install minikube
    brew install kubernetes-helm
    
Then install [Docker for Mac](https://docs.docker.com/docker-for-mac/install/). This contains the HyperKit binary you will need to run Kubernetes on your Mac.

Install the HyperKit driver:

   curl -LO https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-hyperkit \
   && chmod +x docker-machine-driver-hyperkit \
   && sudo mv docker-machine-driver-hyperkit /usr/local/bin/ \
   && sudo chown root:wheel /usr/local/bin/docker-machine-driver-hyperkit \
   && sudo chmod u+s /usr/local/bin/docker-machine-driver-hyperkit

To start Kubernetes:

    minikube start --vm-driver hyperkit --memory 4028
    
Once Kubernetes is started, yoiu need to enable the ingress addon (run once after installation):

    minikube addons enable ingress
    

### Setup Jenkins X

Then ensure that Helm is initialized on your Kubernetes cluster by running:

    helm init
    
Helm will initialize its services on the cluster. You can continue when the `tiller` service is in the `Running` status

    kubectl get pods --namespace kube-system | grep 'tiller'
    tiller-deploy-7594bf7b76-5vh8s          1/1       Running   0          1m

To install Jenkins X:

    jx install
    
On every machine you interact with Jenkins X you will need to run the following command to interact with Jenkins X using the `jx` command line:

    jx init


### Using an existing kubernetes cluster

If you already have a kubernetes cluster setup then try:

    jx install
        
## Opening Consoles

To open the Jenkins console try:

    jx console
    
Or to open other consoles

    jx open foo
    
If you do not know the name

    jx open
    

## Tail logs

To tail the logs of anything running on kubernetes (jenkins or your own applications) type

    jx logs
    
Which prompts you for the deployment to log then tails the logs of the newest pod for an app.

You can filter the list of deployments via:

    jx logs -f cheese

Then if there's only one deployment with a name that contains `cheese` then it'll tail the logs of the latest pod or will prompt you to choose the exact deployment to use.

## Remote shells

You can open a remote shell inside any pods container via the `rsh` command

    jx rsh
    
Or to open a shell inside a pod named foo

    jx rsh foo

Pass `-c` to specify the container name. e.g. to open a shell in a maven build pod

    jx rsh -c maven maven

## Importing or Creating apps

To import an application from the current directory:

    jx import
    
Or to create a new Spring Boot application from scratch:

    jx create spring
    
e.g. to create a new WebMVC and Spring Boot Actuator microservice try this:

    jx create spring -d web -d actuator
        
If you have a Maven Archetype you would like to create then use:

    jx create archetype
    
## Starting builds

To start a pipeline using a specific name try

    jx start pipeline myorg/myrepo

Or to pick the pipeline to start:

    jx start pipeline

If you know part of the name of the pipeline to run you can filter the list via:

    jx start pipeline -f thingy

You can start and tail the build log via:
		
    jx start pipeline -t

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

## Switching Clusters

If you have multiple kubernetes clusters (e.g. you are using GKE and minikube together) then you can switch between them via

    jx ctx
    
In the same way. Or via

    jx ctx minikube
            
**Note** that changing the namespace ,environment or cluster changes the current context for **ALL** shells!

### Sub shells

So if you want to work temporarily with, say, the production cluster we highly recommend you use a sub shell for that.

    jx shell my-prod-context
    
Or to pick the context to use for the sub shell

    jx shell 

Then your bash prompt will be updated to reflect that you are in a different context and/or namespace. Any changes to the namespace, environment or context will be local to the current shell only!    

### Setting your prompt

You can use the `jx prompt` to configure your CLI prompt to display the current team and environment you are working within           
                                            
		# Enable the prompt for bash
		PS1="[\u@\h \W \$(jx prompt)]\$ "

		# Enable the prompt for zsh
		PROMPT='$(jx prompt)'$PROMPT

Note that the prompt is updated automatically for you via the `jx shell` command too

### Bash completion

On a Mac to enable bash completion try:

    jx completion bash > ~/.jx/bash
    source ~/.jx/bash   
    
Or try:

    source <(jx completion bash)

For more help try:

    jx help completion bash
           
### Uninstall Jenkins x

To remove the Jenkins X platfrom from a namespace on your kubernetes cluster:

    jx uninstall

## Addons

We are adding a number of addon capabilities to Jenkins X. To add or remove addons use the `jx create addon` or `jx delete addon` commands

For example to add the [gitea git server](https://gitea.io/en-US/) to your Jenkins X installation try:

    jx create addon gitea

This will: 

* install the gitea helm chart
* add gitea as a git server (via the `jx create git server gitea` command)
* create a new user in gitea (via the `jx create git user -n gitea` command)
* create a new git API token in gitea (via the `jx create git token -n gitea -p password username` command)
     
## Troubleshooting

We have tried to collate common issues here with work arounds. If your issue isn't listed here please [let us know](https://github.com/jenkins-x/jx/issues/new).
 
### Cannot create cluster minikube
If you are using a Mac then `hyperkit` is the best VM driver to use - but does require you to install a recent [Docker for Mac](https://docs.docker.com/docker-for-mac/install/) first. Maybe try that then retry `jx create cluster minikube`?

If your minikube is failing to startup then you could try:
 
    minikube delete
    rm -rf ~/.minikube

If the `rm` fails you may need to do:

    sudo rm -rf ~/.minikube

Now try `jx create cluster minikube` again - did that help? Sometimes there are stale certs or files hanging around from old installations of minikube that can break things.

Sometimes a reboot can help in cases where virtualisation goes wrong ;)

Otherwise you could try follow the minikube instructions 

* [install minikube](https://github.com/kubernetes/minikube#installation)
* [run minikube start](https://github.com/kubernetes/minikube#quickstart) 

### Minkube and hyperkit: Could not find an IP address

If you are using minikube on a mac with hyperkit and find minikube fails to start with a log like:

```
Temporary Error: Could not find an IP address for 46:0:41:86:41:6e
Temporary Error: Could not find an IP address for 46:0:41:86:41:6e
Temporary Error: Could not find an IP address for 46:0:41:86:41:6e
Temporary Error: Could not find an IP address for 46:0:41:86:41:6e
```

It could be you have hit [this issue in minikube and hyperkit](https://github.com/kubernetes/minikube/issues/1926#issuecomment-356378525).

The work around is to try the following:

```
rm ~/.minikube/machines/minikube/hyperkit.pid
``` 

Then try again. Hopefully this time it will work!

### Cannot access services on minikube

When running minikube locally `jx` defaults to using [nip.io](http://nip.io/) as a way of using nice-isn DNS names for services and working around the fact that most laptops can't do wildcard DNS. However sometimes [nip.io](http://nip.io/) has issues and does not work.

To avoid using [nip.io](http://nip.io/) you can do the following:

Edit the file `~/.jenkins-x/cloud-environments/env-minikube/myvalues.yaml` and add the following content:

```yaml
expose:
  Args:
    - --exposer
    - NodePort
    - --http
    - "true"
```

Then re-run `jx install` and this will switch the services to be exposed on `node ports` instead of using ingress and DNS.

So if you type:

```
jx open
```

You'll see all the URs of the form `http://$(minikube ip):somePortNumber` which then avoids going through [nip.io](http://nip.io/) - it just means the URLs are a little more cryptic using magic port numbers rather than simple host names.



 
### Other issues

Please [let us know](https://github.com/jenkins-x/jx/issues/new) and see if we can help? Good luck! 
	
## Contributing

If you're looking to build from source or get started hacking on jx, please see the
[hacking guide][hacking] for more information.


[hacking]: docs/contributing/hacking.md
