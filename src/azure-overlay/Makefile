# Source files common to all targets.
COREFILES = \
	$(wildcard common/*.go) \
	$(wildcard log/*.go) \
	$(wildcard netlink/*.go) \
	$(wildcard network/*.go) \
	$(wildcard platform/*.go) \
	$(wildcard store/*.go)



##TM
OVERLAYFILES = \
		$(wildcard overlay/*.go) \
		$(wildcard overlay/network/*.go) \
		$(wildcard overlay/plugin/*.go) \
		$(COREFILES)
##TM

# Build defaults.
GOOS ?= linux
GOARCH ?= amd64

# Build directories.
OVERLAYDIR = overlay/plugin
##TM

OUTPUT_DIR = output
BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)

# Containerized build parameters.
BUILD_CONTAINER_IMAGE = acn-build
BUILD_CONTAINER_NAME = acn-builder
BUILD_CONTAINER_REPO_PATH = /go/src/azure-overlay
BUILD_USER ?= $(shell id -u)

# TAR file names.

# Docker libnetwork (OVERLAY) plugin image parameters.

##TM
OVERLAY_TAR_NAME = azure-overlay-$(GOOS)-$(GOARCH)-$(VERSION).tgz

# Docker libnetwork (overlay) plugin image parameters.
OVERLAY_PLUGIN_IMAGE = microsoft/azure-overlay
OVRELAY_PLUGIN_ROOTFS = azure-overlay-rootfs

##TM


VERSION ?= $(shell git describe --tags --always --dirty)

ENSURE_OUTPUT_DIR_EXISTS := $(shell mkdir -p $(OUTPUT_DIR))

##TM
azure-overlay: $(BUILD_DIR)/azure-overlay overlay-tar
##TM

##TM
all-binaries: azure-overlay
##TM

# Clean all build artifacts.
.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)
	rm -rf ../../out/linux-amd64/targets/azure-overlay

##TM
$(BUILD_DIR)/azure-overlay: $(OVERLAYFILES)
	go build -v -o $(BUILD_DIR)/azure-overlay -ldflags "-X main.version=$(VERSION) -s -w" $(OVERLAYDIR)/*.go
	mkdir -p ../../out/linux-amd64/targets
	cp $(BUILD_DIR)/azure-overlay ../../out/linux-amd64/targets
##TM

# Build all binaries in a container.
.PHONY: all-binaries-containerized
all-binaries-containerized:
	pwd && ls -l
	docker build -f Dockerfile.build -t $(BUILD_CONTAINER_IMAGE):$(VERSION) .
	docker run --name $(BUILD_CONTAINER_NAME) \
		$(BUILD_CONTAINER_IMAGE):$(VERSION) \
		bash -c '\
			pwd && ls -l && \
			export GOOS=$(GOOS) && \
			export GOARCH=$(GOARCH) && \
			make all-binaries && \
			chown -R $(BUILD_USER):$(BUILD_USER) $(BUILD_DIR) \
		'
	docker cp $(BUILD_CONTAINER_NAME):$(BUILD_CONTAINER_REPO_PATH)/$(BUILD_DIR) $(OUTPUT_DIR)
	docker rm $(BUILD_CONTAINER_NAME)
	docker rmi $(BUILD_CONTAINER_IMAGE):$(VERSION)


##TM
PHONY: azure-overlay-image
azure-overlay-image: azure-overlay
	# Build the plugin image, keeping any old image during build for cache, but remove it afterwards.
	docker images -q $(OVERLAY_PLUGIN_ROOTFS):$(VERSION) > cid
	docker build \
		-f Dockerfile.overlay \
		-t $(OVERLAY_PLUGIN_ROOTFS):$(VERSION) \
		--build-arg BUILD_DIR=$(BUILD_DIR) \
		.
	$(eval CID := `cat cid`)
	docker rmi $(CID) || true

	# Create a container using the image and export its rootfs.
	docker create $(OVERLAY_PLUGIN_ROOTFS):$(VERSION) > cid
	$(eval CID := `cat cid`)
	mkdir -p $(OUTPUT_DIR)/$(CID)/rootfs
	docker export $(CID) | tar -x -C $(OUTPUT_DIR)/$(CID)/rootfs
	docker rm -vf $(CID)

	# Copy the plugin configuration and set ownership.
	cp overlay/config.json $(OUTPUT_DIR)/$(CID)
	chgrp -R docker $(OUTPUT_DIR)/$(CID)

	# Create the plugin.
	docker plugin rm $(OVERLAY_PLUGIN_IMAGE):$(VERSION) || true
	docker plugin create $(OVERLAY_PLUGIN_IMAGE):$(VERSION) $(OUTPUT_DIR)/$(CID)

	# Cleanup temporary files.
	rm -rf $(OUTPUT_DIR)/$(CID)
	rm cid
	# Publish the Azure OVERLAY plugin image to a Docker registry.
.PHONY: publish-azure-overlay-image
publish-azure-overlay-image:
	docker plugin push $(OVERLAY_PLUGIN_IMAGE):$(VERSION)

	# Create a OVERLAY tarball for the current platform.
.PHONY: overlay-tar
overlay-tar:
	chmod 0755 $(BUILD_DIR)/azure-overlay
	cd $(BUILD_DIR) && tar -czvf $(OVERLAY_TAR_NAME) azure-overlay
	chown $(BUILD_USER):$(BUILD_USER) $(BUILD_DIR)/$(OVERLAY_TAR_NAME)
