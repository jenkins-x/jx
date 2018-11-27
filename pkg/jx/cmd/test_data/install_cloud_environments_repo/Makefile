CHART_REPO := http://chartmuseum.jenkins-x.io
CHART := jenkins-x-platform
CHART_VERSION := 0.0.1436
OS := $(shell uname)
HELM := $(shell command -v helm 2> /dev/null)
WATCH := $(shell command -v watch --help 2> /dev/null)
RELEASE := jenkins-x

setup:
	helm repo add jenkins-x $(CHART_REPO)

delete:
	helm delete --purge $(RELEASE)
	kubectl delete cm --all

clean: setup
	rm -rf secrets.yaml.dec
