CHART_REPO := http://jenkins-x-chartmuseum:8080
NAME := controller-build
OS := $(shell uname)
RELEASE_VERSION := $(shell cat ../../pkg/version/VERSION)

setup:
	helm repo add jenkins-x http://chartmuseum.jenkins-x.io

build: setup clean
	helm dependency build
	helm lint

install: package
	helm upgrade --install $(NAME) .

upgrade: package
	helm upgrade --install $(NAME) .

delete:
	helm delete --purge $(NAME)

clean:
	rm -rf charts
	rm -rf ${NAME}*.tgz
	rm -rf requirements.lock

package: setup clean build
ifeq ($(OS),Darwin)
	sed -i "" -e "s/version:.*/version: $(RELEASE_VERSION)/" Chart.yaml
	sed -i "" -e "s/tag:.*/tag: $(RELEASE_VERSION)/" values.yaml

else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(RELEASE_VERSION)/" Chart.yaml
	sed -i -e "s/tag:.*/tag: $(RELEASE_VERSION)/" values.yaml
endif
	helm package .

release: package
	curl --fail -u $(CHARTMUSEUM_CREDS_USR):$(CHARTMUSEUM_CREDS_PSW) --data-binary "@$(NAME)-$(RELEASE_VERSION).tgz" $(CHART_REPO)/api/charts
	rm -rf ${NAME}*.tgz

