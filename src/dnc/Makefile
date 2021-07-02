.DEFAULT_GOAL := all

# Cross-platform directory creation/deletion
MKDIR := mkdir -p
RMDIR := rm -rf
ifeq ($(OS),Windows_NT)
MKDIR := powershell.exe -NoProfile -Command New-Item -ItemType Directory -Force
RMDIR := powershell.exe -NoProfile -Command Remove-Item -Recurse -Force
endif

# Build defaults
GOOS ?= linux
GOARCH ?= amd64
GENERATE_BOND_FILE ?= true

REPO_ROOT := $(shell git rev-parse --show-toplevel)

DNCTL_DIR      = cmd/dnctl
LBRUNNERNC_DIR = test/lbrunnernc
OUTPUT_DIR     = output
REQCTRL_DIR    = cmd/requestcontroller
RUNNER_DIR     = test/runner
TOOLS_DIR      = hack/tools

DNCTL_BUILD_DIR      = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnctl
DNC_BUILD_DIR        = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnc
IMAGE_DIR            = $(OUTPUT_DIR)/images
LBRUNNERNC_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/lbrunnernc
REQCTRL_BUILD_DIR    = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/requestcontroller
RUNNER_BUILD_DIR     = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/runner
TOOLS_BIN_DIR        = $(TOOLS_DIR)/bin
GOLANGCI_LINT        = $(TOOLS_BIN_DIR)/golangci-lint

DNC_AI_ID    = 4a542967-4721-4c64-a2a7-eb9428c634b4
DNC_RP_AI_ID = 6f9518fc-9dc3-492e-87e2-b2b8941a0e92

# Target OS specific parameters
ifeq ($(GOOS),linux)
	# Linux.
	ARCHIVE_CMD = tar -czvf
	ARCHIVE_EXT = tgz
else
	# Windows.
	ARCHIVE_CMD = zip -9lq
	ARCHIVE_EXT = zip
	EXE_EXT = .exe
endif

VERSION = $(shell git describe --tags --always --dirty)

# Archive file names
DNC_ARCHIVE_NAME                = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNC_IMAGE_ARCHIVE_NAME          = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
MDNC_IMAGE_ARCHIVE_NAME         = mdnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNCTL_ARCHIVE_NAME              = dnctl-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
LBRUNNERNC_ARCHIVE_NAME         = lbrunnernc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
LBRUNNERNC_IMAGE_ARCHIVE_NAME   = lbrunnernc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_ARCHIVE_NAME             = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_IMAGE_ARCHIVE_NAME       = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
REQCTRL_ARCHIVE_NAME            = requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
REQCTRL_IMAGE_ARCHIVE_NAME      = requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
MDNC_REQCTRL_ARCHIVE_NAME       = mdnc-requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
MDNC_REQCTRL_IMAGE_ARCHIVE_NAME = mdnc-requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)

# Container image names
DNC_IMAGE          ?= acndev.azurecr.io/dnc
LBRUNNERNC_IMAGE   ?= acndev.azurecr.io/swift-lbrunnernc
RUNNER_IMAGE       ?= acndev.azurecr.io/swift-runner
REQCTRL_IMAGE      ?= acndev.azurecr.io/dnc-request-controller
MDNC_REQCTRL_IMAGE ?= acndev.azurecr.io/mdnc-requestcontroller
MDNC_IMAGE         ?= acndev.azurecr.io/mdnc

# Shorthand target names for convenience.
.PHONY: dnc dncctl lbrunnernc runner requestcontroller
dnc: dnc-binary dnc-archive
dnctl: dnctl-binary dnctl-archive
lbrunnernc: lbrunnernc-binary lbrunnernc-archive
runner: runner-binary runner-archive
requestcontroller: requestcontroller-binary requestcontroller-archive

##@ Building
dnc-binary: export CGO_ENABLED = 0
dnc-binary: ## Build the Azure DNC Service (TODO: enable pubsubproxy)
	go build -mod=vendor -v -o $(DNC_BUILD_DIR)/dnc$(EXE_EXT) -ldflags "-X dnc/utils.Version=$(VERSION) -X dnc/aiwrapper.aiMetadata=$(DNC_AI_ID) -s -w"

dnctl-binary: export CGO_ENABLED = 0
dnctl-binary: ## Build dnctl
	cd $(DNCTL_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT)

lbrunnernc-binary: export CGO_ENABLED = 0
lbrunnernc-binary: ## Build lb runner nc image
	cd $(LBRUNNERNC_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(LBRUNNERNC_BUILD_DIR)/lbrunnernc$(EXE_EXT)

runner-binary: export CGO_ENABLED = 0
runner-binary: ## Build runner
	cd $(RUNNER_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(RUNNER_BUILD_DIR)/runner$(EXE_EXT)

requestcontroller-binary: export CGO_ENABLED = 0
requestcontroller-binary: ## Build Request Controller
	cd $(REQCTRL_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(REQCTRL_BUILD_DIR)/dnc-request-controller$(EXE_EXT)

all: dnc dnctl lbrunnernc runner requestcontroller ## Build all binaries

##@ Testing
.PHONY: lint
LINT_PKG ?= .

test: ## Run tests
	go test ./...

lint: $(GOLANGCI_LINT) ## Fast lint
	$(GOLANGCI_LINT) run -v $(LINT_PKG)/...

##@ Containerizing
dnc-image: ## Build the Azure DNC image
	docker build --no-cache \
	-f $(REPO_ROOT)/src/dnc/builds/dnc/Dockerfile \
	-t $(DNC_IMAGE):$(VERSION) \
	--build-arg GENERATE_BOND_FILE=$(GENERATE_BOND_FILE) \
	--build-arg VERSION=$(VERSION) \
	$(REPO_ROOT)
	$(MKDIR) $(IMAGE_DIR)
	docker save $(DNC_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(DNC_IMAGE_ARCHIVE_NAME)

mdnc-image: ##
	docker build --no-cache \
	-f $(REPO_ROOT)/src/dnc/builds/dnc/Dockerfile \
	-t $(MDNC_IMAGE):$(VERSION) \
	--build-arg aiMetadata=$(DNC_RP_AI_ID) \
	--build-arg GENERATE_BOND_FILE=$(GENERATE_BOND_FILE) \
	--build-arg VERSION=$(VERSION) \
	$(REPO_ROOT)
	$(MKDIR) $(IMAGE_DIR)
	docker save $(MDNC_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(MDNC_IMAGE_ARCHIVE_NAME)

requestcontroller-image: ## Build the requstcontroller image
	docker build --no-cache \
	-f builds/requestcontroller/Dockerfile \
	-t $(REQCTRL_IMAGE):$(VERSION) \
	--build-arg VERSION=$(VERSION) \
	.
	$(MKDIR) $(IMAGE_DIR)
	docker save $(REQCTRL_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(REQCTRL_IMAGE_ARCHIVE_NAME)

mdnc-requestcontroller-image: ## Build the requstcontroller image
	docker build --no-cache \
	-f builds/requestcontroller/Dockerfile \
	-t $(MDNC_REQCTRL_IMAGE):$(VERSION) \
	--build-arg aiMetadata=$(DNC_RP_AI_ID) \
	--build-arg VERSION=$(VERSION) \
	.
	$(MKDIR) $(IMAGE_DIR)
	docker save $(MDNC_REQCTRL_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(MDNC_REQCTRL_IMAGE_ARCHIVE_NAME)

runner-image: runner-binary ## Build the runner image
	docker build --no-cache \
	-f builds/runner/Dockerfile \
	-t $(RUNNER_IMAGE):$(VERSION) \
	.
	$(MKDIR) $(IMAGE_DIR)
	docker save $(RUNNER_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(RUNNER_IMAGE_ARCHIVE_NAME)

lbrunnernc-image: lbrunnernc-binary ## Build the lb runner nc image (TODO: multi-stage docker build)
ifeq ($(GOOS),linux)
	docker build --no-cache \
	-f test/lbrunnernc/Dockerfile \
	-t $(LBRUNNERNC_IMAGE):$(VERSION) \
	--build-arg LBRUNNERNC_BUILD_DIR=$(LBRUNNERNC_BUILD_DIR) \
	.
	$(MKDIR) $(IMAGE_DIR)
	docker save $(LBRUNNERNC_IMAGE):$(VERSION) | gzip -c > $(IMAGE_DIR)/$(LBRUNNERNC_IMAGE_ARCHIVE_NAME)
endif

##@ Archives
.PHONY: dnc-archive dnctl-archive lbrunnernc-archive runner-archive requestcontroller-archive

$(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME): ## Create a dnc archive for the target platform
	cd $(DNC_BUILD_DIR) && $(ARCHIVE_CMD) $(DNC_ARCHIVE_NAME) dnc$(EXE_EXT)

dnc-archive: $(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME) ##

$(DNCTL_BUILD_DIR)/$(DNCTL_ARCHIVE_NAME): ## Create a dnctl archive for the target platform
	cd $(DNCTL_BUILD_DIR) && $(ARCHIVE_CMD) $(DNCTL_ARCHIVE_NAME) dnctl$(EXE_EXT)

dnctl-archive: $(DNCTL_BUILD_DIR)/$(DNCTL_ARCHIVE_NAME) ##

$(LBRUNNERNC_BUILD_DIR)/$(LBRUNNERNC_ARCHIVE_NAME): ## Create a lb runner nc archive for the target platform
	cd $(LBRUNNERNC_BUILD_DIR) && $(ARCHIVE_CMD) $(LBRUNNERNC_ARCHIVE_NAME) lbrunnernc$(EXE_EXT)

lbrunnernc-archive: $(LBRUNNERNC_BUILD_DIR)/$(LBRUNNERNC_ARCHIVE_NAME) ##

$(RUNNER_BUILD_DIR)/$(RUNNER_ARCHIVE_NAME): ## Create a runner archive for the target platform
	cd $(RUNNER_BUILD_DIR) && $(ARCHIVE_CMD) $(RUNNER_ARCHIVE_NAME) runner$(EXE_EXT)

runner-archive: $(RUNNER_BUILD_DIR)/$(RUNNER_ARCHIVE_NAME) ##

$(REQCTRL_BUILD_DIR)/$(REQCTRL_ARCHIVE_NAME): ## Create a requetscontroller archive for the target platform
	cd $(REQCTRL_BUILD_DIR) && $(ARCHIVE_CMD) $(REQCTRL_ARCHIVE_NAME) dnc-request-controller$(EXE_EXT)

requestcontroller-archive: $(REQCTRL_BUILD_DIR)/$(REQCTRL_ARCHIVE_NAME) ##

##@ Publishing
publish-dnc-image: ## Publish the Azure dnc image to a Docker registry
	docker push $(DNC_IMAGE):$(VERSION)

publish-lbrunnernc-image: ## Publish the lb runner nc image to a Docker registry
	docker push $(LBRUNNERNC_IMAGE):$(VERSION)

publish-runner-image: ## Publish the runner image to a Docker registry
	docker push $(RUNNER_IMAGE):$(VERSION)


##@ Tasks 
.PHONY: clean dev help

$(TOOLS_DIR)/go.mod: 
	cd $(TOOLS_DIR);go mod init && go mod tidy

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Build golangci-lint
	cd $(TOOLS_DIR); go build -tags=tools -o bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

clean: ## Clean all build artifacts
	$(RMDIR) $(OUTPUT_DIR)

dev: ## Create a dev environment to step through debug in an AKS cluster using Skaffold
	skaffold dev -f hack/skaffold.yaml --port-forward

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
