# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# exclude python from std builds
DIRS=`go list ./... | grep -v python`

all: build

build: 
	@echo "GO111MODULE = $(value GO111MODULE)"
	$(GOBUILD) -v $(DIRS)

test: 
	@echo "GO111MODULE = $(value GO111MODULE)"
	$(GOTEST) -v $(DIRS)

clean: 
	@echo "GO111MODULE = $(value GO111MODULE)"
	$(GOCLEAN) ./...

vet:
	@echo "GO111MODULE = $(value GO111MODULE)"
	$(GOCMD) vet $(DIRS) | grep -v unkeyed

tidy: export GO111MODULE = on
tidy:
	@echo "GO111MODULE = $(value GO111MODULE)"
	go mod tidy
	
# updates go.mod to master for all of the goki dependencies
# note: must somehow remember to do this for any other depend
# that we need to update at any point!
master: export GO111MODULE = on
master:
	@echo "GO111MODULE = $(value GO111MODULE)"
	go get -u github.com/goki/ki@master
	go get -u github.com/goki/pi@master
	go get -u github.com/goki/gi@master
	go list -m all | grep goki
	
old:
	@echo "GO111MODULE = $(value GO111MODULE)"
	go list -u -m all | grep '\['
	
update:
	@echo "GO111MODULE = $(value GO111MODULE)"
	go get -u ./...
	go mod tidy

# gopath-update is for GOPATH to get most things updated.
# need to call it in a target executable directory
gopath-update: export GO111MODULE = off
gopath-update:
	@echo "GO111MODULE = $(value GO111MODULE)"
	cd cmd/gide; go get -u ./...
	
release:
	$(MAKE) -C gide release

