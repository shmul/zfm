PROJ_NAME=zfm

 # Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=$(PROJ_NAME)
BINARY_LINUX=$(BINARY_NAME)_linux
HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%dT%H:%M:%S')
BRANCH := $(shell git branch --show-current)
PVARS := main
LDFLAGS := "-X '${PVARS}.Branch=${BRANCH}' -X '${PVARS}.Timestamp=${BUILD_DATE}' -X '${PVARS}.Revision=${HASH}'"
LINUX_FLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=amd64

ifeq ($(VERBOSE),1)
  quiet =
  Q =
else
  quiet=quiet_
  Q = @
endif

.PHONY: clean test build all fmt

all: build

build:
	$(Q) $(GOBUILD) -o $(BINARY_NAME) -ldflags=$(LDFLAGS) ./cmd/zfm/

# Cross compilation
build-linux:
	$(Q) $(LINUX_FLAGS) $(GOBUILD) -o $(BINARY_LINUX) -ldflags=$(LDFLAGS) ./cmd/zfm/

test:
	$(Q) $(GOTEST) -race ./...

fmt:
	$(Q) $(GOCMD) fmt ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_LINUX)
