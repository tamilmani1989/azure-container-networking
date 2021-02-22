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

DNCTL_DIR = cmd/dnctl
LBRUNNERNC_DIR = test/lbrunnernc
RUNNER_DIR = test/runner
REQCTRL_DIR = cmd/requestcontroller
OUTPUT_DIR = output
DNC_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnc
DNCTL_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnctl
LBRUNNERNC_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/lbrunnernc
RUNNER_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/runner
REQCTRL_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/requestcontroller
REPO_ROOT := $(shell git rev-parse --show-toplevel)

DNC_AI_ID = 4a542967-4721-4c64-a2a7-eb9428c634b4
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
DNC_ARCHIVE_NAME = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNC_IMAGE_ARCHIVE_NAME = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
MDNC_IMAGE_ARCHIVE_NAME = mdnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNCTL_ARCHIVE_NAME = dnctl-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
LBRUNNERNC_ARCHIVE_NAME = lbrunnernc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
LBRUNNERNC_IMAGE_ARCHIVE_NAME = lbrunnernc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_ARCHIVE_NAME = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_IMAGE_ARCHIVE_NAME = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
REQCTRL_ARCHIVE_NAME = requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
REQCTRL_IMAGE_ARCHIVE_NAME = requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)

# Container image names
DNC_IMAGE = containernetworking.azurecr.io/internal/containernetworking/dnc
LBRUNNERNC_IMAGE = containernetworking.azurecr.io/public/containernetworking/swift-lbrunnernc
RUNNER_IMAGE = containernetworking.azurecr.io/public/containernetworking/swift-runner
REQCTRL_IMAGE = containernetworking.azurecr.io/internal/containernetworking/requestcontroller
MDNC_IMAGE = ancdev.azurecr.io/mdnc

# Default target is "all"
.PHONY: all
all: dnc dnctl lbrunnernc runner requestcontroller

# Shorthand target names for convenience.
dnc: dnc-binary dnc-archive
dnctl: dnctl-binary dnctl-archive
lbrunnernc: lbrunnernc-binary lbrunnernc-archive
runner: runner-binary runner-archive
requestcontroller: requestcontroller-binary requestcontroller-archive

# Clean all build artifacts
.PHONY: clean
clean:
	$(RMDIR) $(OUTPUT_DIR)

########################### Binaries ###########################

# Build the Azure DNC Service (TODO: enable pubsubproxy)
.PHONY: dnc-binary
dnc-binary: export CGO_ENABLED = 0
dnc-binary:
	go build -mod=vendor -v -o $(DNC_BUILD_DIR)/dnc$(EXE_EXT) -ldflags "-X dnc/utils.Version=$(VERSION) -X dnc/aiwrapper.aiMetadata=$(DNC_AI_ID) -s -w"

# Build dnctl
.PHONY: dnctl-binary
dnctl-binary: export CGO_ENABLED = 0
dnctl-binary:
	cd $(DNCTL_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT)

# Build lb runner nc image
.PHONY: lbrunnernc-binary
lbrunnernc-binary: export CGO_ENABLED = 0
lbrunnernc-binary:
	cd $(LBRUNNERNC_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(LBRUNNERNC_BUILD_DIR)/lbrunnernc$(EXE_EXT)

# Build runner
.PHONY: runner-binary
runner-binary: export CGO_ENABLED = 0
runner-binary:
	cd $(RUNNER_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(RUNNER_BUILD_DIR)/runner$(EXE_EXT)

# Build Request Controller
.PHONY: requestcontroller-binary
requestcontroller-binary: export CGO_ENABLED = 0
requestcontroller-binary:
	cd $(REQCTRL_DIR) && go build -mod=vendor -v -o $(REPO_ROOT)/src/dnc/$(REQCTRL_BUILD_DIR)/dnc-request-controller$(EXE_EXT)

########################### Container Images ###########################

# Build the Azure DNC image
.PHONY: dnc-image
dnc-image:
	docker build --no-cache \
	-f $(REPO_ROOT)/src/dnc/builds/dnc/Dockerfile \
	-t $(DNC_IMAGE):$(VERSION) \
	$(REPO_ROOT)
	$(MKDIR) $(DNC_BUILD_DIR)
	docker save $(DNC_IMAGE):$(VERSION) | gzip -c > $(DNC_BUILD_DIR)/$(DNC_IMAGE_ARCHIVE_NAME)

.PHONY: mdnc-image
mdnc-image:
	docker build --no-cache \
	-f $(REPO_ROOT)/src/dnc/builds/dnc/Dockerfile \
	-t $(MDNC_IMAGE):$(VERSION) \
	--build-arg aiMetadata=$(DNC_RP_AI_ID) \
	$(REPO_ROOT)
	$(MKDIR) $(DNC_BUILD_DIR)
	docker save $(MDNC_IMAGE):$(VERSION) | gzip -c > $(DNC_BUILD_DIR)/$(MDNC_IMAGE_ARCHIVE_NAME)

# Build the requstcontroller image
.PHONY: requestcontroller-image
requestcontroller-image:
	docker build --no-cache \
	-f builds/requestcontroller/Dockerfile \
	-t $(REQCTRL_IMAGE):$(VERSION) \
	--build-arg aiMetadata=$(DNC_RP_AI_ID) \
	--build-arg VERSION=$(VERSION) \
	.
	$(MKDIR) $(REQCTRL_BUILD_DIR)
	docker save $(REQCTRL_IMAGE):$(VERSION) | gzip -c > $(REQCTRL_BUILD_DIR)/$(REQCTRL_IMAGE_ARCHIVE_NAME)

# Build the runner image
.PHONY: runner-image
runner-image: runner-binary
	docker build --no-cache \
	-f builds/runner/Dockerfile \
	-t $(RUNNER_IMAGE):$(VERSION) \
	.
	$(MKDIR) $(RUNNER_BUILD_DIR)
	docker save $(RUNNER_IMAGE):$(VERSION) | gzip -c > $(RUNNER_BUILD_DIR)/$(RUNNER_IMAGE_ARCHIVE_NAME)

# Build the lb runner nc image (TODO: multi-stage docker build)
.PHONY: lbrunnernc-image
lbrunnernc-image: lbrunnernc-binary
ifeq ($(GOOS),linux)
	docker build --no-cache \
	-f test/lbrunnernc/Dockerfile \
	-t $(LBRUNNERNC_IMAGE):$(VERSION) \
	--build-arg LBRUNNERNC_BUILD_DIR=$(LBRUNNERNC_BUILD_DIR) \
	.
	$(MKDIR) $(LBRUNNERNC_BUILD_DIR)
	docker save $(LBRUNNERNC_IMAGE):$(VERSION) | gzip -c > $(LBRUNNERNC_BUILD_DIR)/$(LBRUNNERNC_IMAGE_ARCHIVE_NAME)
endif

########################### Archives ###########################

# Create a dnc archive for the target platform
$(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME):
	cd $(DNC_BUILD_DIR) && $(ARCHIVE_CMD) $(DNC_ARCHIVE_NAME) dnc$(EXE_EXT)

.PHONY: dnc-archive
dnc-archive: $(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME)

# Create a dnctl archive for the target platform
$(DNCTL_BUILD_DIR)/$(DNCTL_ARCHIVE_NAME):
	cd $(DNCTL_BUILD_DIR) && $(ARCHIVE_CMD) $(DNCTL_ARCHIVE_NAME) dnctl$(EXE_EXT)

.PHONY: dnctl-archive
dnctl-archive: $(DNCTL_BUILD_DIR)/$(DNCTL_ARCHIVE_NAME)

# Create a lb runner nc archive for the target platform
$(LBRUNNERNC_BUILD_DIR)/$(LBRUNNERNC_ARCHIVE_NAME):
	cd $(LBRUNNERNC_BUILD_DIR) && $(ARCHIVE_CMD) $(LBRUNNERNC_ARCHIVE_NAME) lbrunnernc$(EXE_EXT)

.PHONY: lbrunnernc-archive
lbrunnernc-archive: $(LBRUNNERNC_BUILD_DIR)/$(LBRUNNERNC_ARCHIVE_NAME)

# Create a runner archive for the target platform
$(RUNNER_BUILD_DIR)/$(RUNNER_ARCHIVE_NAME):
	cd $(RUNNER_BUILD_DIR) && $(ARCHIVE_CMD) $(RUNNER_ARCHIVE_NAME) runner$(EXE_EXT)

.PHONY: runner-archive
runner-archive: $(RUNNER_BUILD_DIR)/$(RUNNER_ARCHIVE_NAME)

# Create a requetscontroller archive for the target platform
$(REQCTRL_BUILD_DIR)/$(REQCTRL_ARCHIVE_NAME):
	cd $(REQCTRL_BUILD_DIR) && $(ARCHIVE_CMD) $(REQCTRL_ARCHIVE_NAME) dnc-request-controller$(EXE_EXT)

.PHONY: requestcontroller-archive
requestcontroller-archive: $(REQCTRL_BUILD_DIR)/$(REQCTRL_ARCHIVE_NAME)

########################### Tasks ###########################
# Publish the Azure dnc image to a Docker registry
.PHONY: publish-dnc-image
publish-dnc-image:
	docker push $(DNC_IMAGE):$(VERSION)

# Publish the lb runner nc image to a Docker registry
.PHONY: publish-lbrunnernc-image
publish-lbrunnernc-image:
	docker push $(LBRUNNERNC_IMAGE):$(VERSION)

# Publish the runner image to a Docker registry
.PHONY: publish-runner-image
publish-runner-image:
	docker push $(RUNNER_IMAGE):$(VERSION)

# Create a dev environment to step through debug in an AKS cluster using Skaffold
.PHONY: dev
dev:
	skaffold dev -f hack/skaffold.yaml --port-forward
