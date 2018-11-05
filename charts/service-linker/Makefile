OS := $(shell uname)

service-linker:
	helm init --client-only
	helm repo add chartmuseum http://jenkins-x-chartmuseum:8080
	helm repo add chartmuseum http://chartmuseum.jenkins-x.io
ifeq ($(OS),Darwin)
	sed -i "" -e "s/version:.*/version: $(PREVIEW_VERSION)/" Chart.yaml
	sed -i "" -e "s/version:.*/version: $(PREVIEW_VERSION)/" ../*/Chart.yaml
	sed -i "" -e "s/tag: .*/tag: $(PREVIEW_VERSION)/" values.yaml
else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(PREVIEW_VERSION)/" Chart.yaml
	sed -i -e "s/version:.*/version: $(PREVIEW_VERSION)/" ../*/Chart.yaml
	sed -i -e "s/repository: .*/repository: $(JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST):$(JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT)\/jenkins-x-bot\/service-linker/" values.yaml
	sed -i -e "s/tag: .*/tag: $(PREVIEW_VERSION)/" values.yaml
else
	echo "platform $(OS) not supported to release from"
	exit -1
endif
	echo "  version: $(PREVIEW_VERSION)" >> requirements.yaml
	helm dependency build
	helm lint
