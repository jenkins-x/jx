CHART_REPO := http://jenkins-x-chartmuseum:8080
NAME := jx
OS := $(shell uname)
GITHUB_ACCESS_TOKEN := $(shell cat /builder/home/git-token 2> /dev/null)
CHARTMUSEUM_CREDS_USR := $(shell cat /builder/home/basic-auth-user 2> /dev/null)
CHARTMUSEUM_CREDS_PSW := $(shell cat /builder/home/basic-auth-pass 2> /dev/null)

setup:
	helm repo add jenkins-x http://chartmuseum.jenkins-x.io

build: setup clean
	helm dependency build
	helm lint

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
	sed -i "" -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i "" -e "s/tag:.*/tag: $(VERSION)/" values.yaml

else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i -e "s/tag:.*/tag: $(VERSION)/" values.yaml
endif

	helm package .
	curl --fail -u $(CHARTMUSEUM_CREDS_USR):$(CHARTMUSEUM_CREDS_PSW) --data-binary "@$(NAME)-$(VERSION).tgz" $(CHART_REPO)/api/charts
	rm -rf ${NAME}*.tgz

