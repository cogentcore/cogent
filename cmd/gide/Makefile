# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get


all: build

build: 
	$(GOBUILD) -v
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
app-install:
	/bin/cp -av gide.app /Applications
mac-sh-install:
	/bin/cp gide-mac.sh /usr/local/bin/gide

