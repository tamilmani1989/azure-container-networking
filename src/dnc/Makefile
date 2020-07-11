# Source files for building DNC plugin
DNCFILES = \
	$(wildcard *.go) \
	$(wildcard aiwrapper/*.go) \
	$(wildcard api/*.go) \
	$(wildcard configuration/*.go) \
	$(wildcard controller/*.go) \
	$(wildcard datamodel/*.go) \
	$(wildcard datastore/*.go) \
	$(wildcard datastoreshim/*.go) \
	$(wildcard logger/*.go) \
	$(wildcard masterelection/*.go) \
	$(wildcard nmagentcontracts/*.go) \
	$(wildcard replica/*.go) \
	$(wildcard server/*.go) \
	$(wildcard utils/*.go)

# Source files for building runner
RUNNERFILES = \
	$(wildcard test/runner/*.go)

# Build defaults
GOOS ?= linux
GOARCH ?= amd64

DNCTL_DIR = cmd/dnctl
RUNNER_DIR = test/runner
REQCTRL_DIR = cmd/requestcontroller
OUTPUT_DIR = output
DNC_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnc
DNCTL_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/dnctl
RUNNER_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/runner
REQCTRL_BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)/requestcontroller

DNC_AI_ID = 4a542967-4721-4c64-a2a7-eb9428c634b4

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

#VERSION ?= $(shell git describe --tags --always --dirty)
VERSION = latest

ENSURE_OUTPUT_DIR_EXISTS := $(shell mkdir -p $(OUTPUT_DIR))

# Archive file names.
DNC_ARCHIVE_NAME = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNCTL_ARCHIVE_NAME = dnctl-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_ARCHIVE_NAME = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
DNC_IMAGE_ARCHIVE_NAME = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
RUNNER_IMAGE_ARCHIVE_NAME = runner-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
REQCTRL_ARCHIVE_NAME = requestcontroller-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)

# Azure container networking service image paramters
DNC_IMAGE = containernetworking.azurecr.io/internal/containernetworking/dnc
RUNNER_IMAGE = containernetworking.azurecr.io/public/containernetworking/swift-runner
REQCTRL_IMAGE = containernetworking.azurecr.io/internal/containernetworking/requestcontroller

# Shorthand target names for convenience.
dnc: $(DNC_BUILD_DIR)/dnc$(EXE_EXT) dnc-archive
dnctl: $(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT) dnctl-archive
runner: $(RUNNER_BUILD_DIR)/runner$(EXE_EXT) runner-archive
requestcontroller: $(REQCTRL_BUILD_DIR)/requestcontroller$(EXE_EXT) requestcontroller-archive

# Clean all build artifacts
.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)

# Build the Azure DNC Service
$(DNC_BUILD_DIR)/dnc$(EXE_EXT): $(DNCFILES)
	export CGO_ENABLED=0; \
	go build -v -o $(DNC_BUILD_DIR)/dnc$(EXE_EXT) -ldflags "-X dnc/utils.Version=$(VERSION) -X dnc/aiwrapper.aiMetadata=$(DNC_AI_ID) -s -w" *.go

# Build dnctl
$(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT): $(DNCTLFILES)
	export CGO_ENABLED=0; \
	go build -v -o $(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT) $(DNCTL_DIR)/*.go

# Build runner
$(RUNNER_BUILD_DIR)/runner$(EXE_EXT): $(RUNNERFILES)
	export CGO_ENABLED=0; \
	go build -v -o $(RUNNER_BUILD_DIR)/runner$(EXE_EXT) $(RUNNER_DIR)/*.go

.PHONY: $(REQCTRL_BUILD_DIR)/requestcontroller$(EXE_EXT)
# Build Request Controller
$(REQCTRL_BUILD_DIR)/requestcontroller$(EXE_EXT):
	export CGO_ENABLED=0; \
	go build -v -o $(REQCTRL_BUILD_DIR)/requestcontroller$(EXE_EXT) $(REQCTRL_DIR)/*.go

# Build the Azure DNC image
.PHONY: dnc-image
dnc-image: dnc
ifeq ($(GOOS),linux)
	docker build \
	-f Dockerfile \
	-t $(DNC_IMAGE):$(VERSION) \
	--build-arg DNC_BUILD_DIR=$(DNC_BUILD_DIR) \
	--network=host \
	.
	docker save $(DNC_IMAGE):$(VERSION) | gzip -c > $(DNC_BUILD_DIR)/$(DNC_IMAGE_ARCHIVE_NAME)
endif

# Build the runner image
.PHONY: runner-image
runner-image: runner
ifeq ($(GOOS),linux)
	docker build \
	-f test/runner/Dockerfile \
	-t $(RUNNER_IMAGE):$(VERSION) \
	--build-arg RUNNER_BUILD_DIR=$(RUNNER_BUILD_DIR) \
	--network=host \
	.
	docker save $(RUNNER_IMAGE):$(VERSION) | gzip -c > $(RUNNER_BUILD_DIR)/$(RUNNER_IMAGE_ARCHIVE_NAME)
endif

# Publish the Azure dnc image to a Docker registry
.PHONY: publish-dnc-image
publish-dnc-image:
	docker push $(DNC_IMAGE):$(VERSION)

# Publish the runner image to a Docker registry
.PHONY: publish-runner-image
publish-runner-image:
	docker push $(RUNNER_IMAGE):$(VERSION)

# Create a dnc archive for the target platform
.PHONY: dnc-archive
dnc-archive:
	chmod 0755 $(DNC_BUILD_DIR)/dnc$(EXE_EXT)
	cd $(DNC_BUILD_DIR) && $(ARCHIVE_CMD) $(DNC_ARCHIVE_NAME) dnc$(EXE_EXT)
	chown $(BUILD_USER):$(BUILD_USER) $(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME)

# Create a dnctl archive for the target platform
.PHONY: dnctl-archive
dnctl-archive:
	chmod 0755 $(DNCTL_BUILD_DIR)/dnctl$(EXE_EXT)
	cd $(DNCTL_BUILD_DIR) && $(ARCHIVE_CMD) $(DNCTL_ARCHIVE_NAME) dnctl$(EXE_EXT)
	chown $(BUILD_USER):$(BUILD_USER) $(DNCTL_BUILD_DIR)/$(DNCTL_ARCHIVE_NAME)

# Create a runner archive for the target platform
.PHONY: runner-archive
runner-archive:
	chmod 0755 $(RUNNER_BUILD_DIR)/runner$(EXE_EXT)
	cd $(RUNNER_BUILD_DIR) && $(ARCHIVE_CMD) $(RUNNER_ARCHIVE_NAME) runner$(EXE_EXT)
	chown $(BUILD_USER):$(BUILD_USER) $(RUNNER_BUILD_DIR)/$(RUNNER_ARCHIVE_NAME)

# Create a requetscontroller archive for the target platform
.PHONY: requestcontroller-archive
requestcontroller-archive:
	chmod 0755 $(REQCTRL_BUILD_DIR)/requestcontroller$(EXE_EXT)
	cd $(REQCTRL_BUILD_DIR) && $(ARCHIVE_CMD) $(REQCTRL_ARCHIVE_NAME) requestcontroller$(EXE_EXT)
	chown $(BUILD_USER):$(BUILD_USER) $(REQCTRL_BUILD_DIR)/$(REQCTRL_ARCHIVE_NAME)