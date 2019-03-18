CHART_REPO := http://jenkins-x-chartmuseum:8080
NAME := jx
OS := $(shell uname)
CHARTMUSEUM_CREDS_USR := $(shell cat /builder/home/basic-auth-user 2> /dev/null)
CHARTMUSEUM_CREDS_PSW := $(shell cat /builder/home/basic-auth-pass 2> /dev/null)

setup:
	helm repo add jenkins-x http://chartmuseum.jenkins-x.io

build: setup clean
	helm dependency build
	helm lint charts/jx

install: clean setup build
	helm upgrade --install $(NAME) .

upgrade: clean setup build
	helm upgrade --install $(NAME) .

delete:
	helm delete --purge $(NAME)

clean: 
	rm -rf charts
	rm -rf ${NAME}*.tgz
	rm -rf requirements.lock

release: setup clean build

ifeq ($(OS),Darwin)
	sed -i "" -e "s/version:.*/version: $(VERSION)/" charts/jx/Chart.yaml
	sed -i "" -e "s/tag:.*/tag: $(VERSION)/" charts/jx/values.yaml

else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(VERSION)/" charts/jx/Chart.yaml
	sed -i -e "s/tag:.*/tag: $(VERSION)/" charts/jx/values.yaml
endif

	helm package .
	curl --fail -u $(CHARTMUSEUM_CREDS_USR):$(CHARTMUSEUM_CREDS_PSW) --data-binary "@$(NAME)-$(VERSION).tgz" $(CHART_REPO)/api/charts
	rm -rf ${NAME}*.tgz