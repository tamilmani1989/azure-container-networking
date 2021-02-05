REPO_ROOT := $(shell git rev-parse --show-toplevel)
PUSH_TO_MINIKUBE_CMD := $(REPO_ROOT)/src/dnc/requestcontroller/hack/scripts/push_to_minikube.sh
ifeq ($(OS),Windows_NT)
PUSH_TO_MINIKUBE_CMD := powershell.exe -NoProfile -Command $(REPO_ROOT)/src/dnc/requestcontroller/hack/scripts/push_to_minikube.ps1
endif
HELM_CHART_DIR=$(REPO_ROOT)/src/dnc/requestcontroller/hack/charts
MINKUBE_START_OPTS = --cni=flannel
SKIP_UPLOAD_IMAGES ?= false

# TODO: Is there a better way to skip uploading images to minikube cluster?
ifeq ($(SKIP_UPLOAD_IMAGES),true)
.PHONY: deploy-swift-local
deploy-swift-local: start-env-local
	helm upgrade swiftwithmocks $(HELM_CHART_DIR)/swift-with-mocks/ --install
else
.PHONY: deploy-swift-local
deploy-swift-local: start-env-local upload-all-local
	helm upgrade swiftwithmocks $(HELM_CHART_DIR)/swift-with-mocks/ --install
endif

.PHONY: start-env-local
start-env-local:
	minikube start $(MINKUBE_START_OPTS)

.PHONY: upload-all-local
upload-all-local: upload-dnc-local upload-cnsmock-local upload-nmagentmock-local upload-dnctl-local

.PHONY: upload-dnc-local
upload-dnc-local:
	make -C $(REPO_ROOT)/src/dnc dnc-image
	$(PUSH_TO_MINIKUBE_CMD) containernetworking.azurecr.io/internal/containernetworking/dnc dnc:latest

.PHONY: cns-mock-image
cns-mock-image:
	docker build --no-cache \
	-f $(REPO_ROOT)/src/dnc/builds/cnsmock/Dockerfile \
	-t cnsmock:latest \
	$(REPO_ROOT)/src/dnc/test/cnsmock

.PHONY: upload-cnsmock-local
upload-cnsmock-local: cns-mock-image
	$(PUSH_TO_MINIKUBE_CMD) cnsmock:latest cnsmock:latest

.PHONY: nmagent-mock-image
nmagent-mock-image:
	docker build -t nmagentmock:latest -f $(REPO_ROOT)/src/dnc/builds/nmagentmock/Dockerfile $(REPO_ROOT)/src/dnc

.PHONY: upload-nmagentmock-local
upload-nmagentmock-local: nmagent-mock-image
	$(PUSH_TO_MINIKUBE_CMD) nmagentmock:latest nmagentmock:latest

.PHONY: dnctl-image
dnctl-image:
	docker build -t dnctl:latest -f $(REPO_ROOT)/src/dnc/builds/dnctl/Dockerfile $(REPO_ROOT)/src/dnc

.PHONY: upload-dnctl-local
upload-dnctl-local: dnctl-image
	$(PUSH_TO_MINIKUBE_CMD) dnctl:latest dnctl:latest

.PHONY: clean-local
clean-local:
	minikube delete