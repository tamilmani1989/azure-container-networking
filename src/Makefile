# Source files for dnc
DNCFILES = \
	$(wildcard dnc/allocationhelper/*.go) \
	$(wildcard dnc/api/*.go) \
	$(wildcard dnc/configuration/*.go) \
	$(wildcard dnc/controller/*.go) \
	$(wildcard dnc/datamodel/*.go) \
	$(wildcard dnc/datastore/*/*.go) \
	$(wildcard dnc/datastoreshim/*.go) \
	$(wildcard dnc/logger/*/*.go) \
	$(wildcard dnc/masterelection/*/.go) \
	$(wildcard dnc/nmagentcontracts/*.go) \
	$(wildcard dnc/replica/*.go) \
	$(wildcard dnc/server/*.go) \
	$(wildcard dnc/utils/*.go)
	


# Build defaults.
GOOS ?= linux
GOARCH ?= amd64



# Build directories.
DNC_DIR = dnc
OUTPUT_DIR = Build
BUILD_DIR = $(OUTPUT_DIR)/$(GOOS)_$(GOARCH)
DNC_BUILD_DIR = $(BUILD_DIR)/dnc



# Containerized build parameters.
BUILD_CONTAINER_IMAGE = dnc-build
BUILD_CONTAINER_NAME = dnc-builder
BUILD_USER ?= $(shell id -u)



# Target OS specific parameters.
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



# Archive file names.
DNC_ARCHIVE_NAME = dnc-$(GOOS)-$(GOARCH)-$(VERSION).$(ARCHIVE_EXT)
VERSION ?= $(shell git describe --tags --always --dirty)
ENSURE_OUTPUT_DIR_EXISTS := $(shell mkdir -p $(OUTPUT_DIR))



# Shorthand target names for convenience.
dnc: $(DNC_BUILD_DIR)/azure-dnc$(EXE_EXT) dnc-archive
all-binaries: dnc



# Clean all build artifacts.
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)



# Build the DNC.
$(DNC_BUILD_DIR)/azure-dnc$(EXE_EXT): $(DNCFILES)
	go build -v -o $(DNC_BUILD_DIR)/azure-dnc$(EXE_EXT) -ldflags "-X main.version=$(VERSION) -s -w" $(DNC_DIR)/*.go




# Create a dnc archive for the target platform.
.PHONY: dnc-archive
dnc-archive:
	chmod 0755 $(DNC_BUILD_DIR)/azure-dnc$(EXE_EXT)

	cd $(DNC_BUILD_DIR) && $(ARCHIVE_CMD) $(DNC_ARCHIVE_NAME) azure-dnc$(EXE_EXT)

	chown $(BUILD_USER):$(BUILD_USER) $(DNC_BUILD_DIR)/$(DNC_ARCHIVE_NAME)
