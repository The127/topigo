GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=topico

all: build

build:
      $(GOGET) -u ./...
      $(GOBUILD) -o $(BINARY_NAME) -v