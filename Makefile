GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=topigo

all: build

build:
	$(GOGET) -u github.com/nanobox-io/golang-scribble
	$(GOGET) -u github.com/kelseyhightower/envconfig
	$(GOGET) -u gopkg.in/go-playground/validator.v9
	$(GOGET) -u github.com/rs/xid
	$(GOGET) -u github.com/golang/protobuf/proto
	$(GOGET) -u google.golang.org/protobuf/reflect/protoreflect
	$(GOGET) -u google.golang.org/protobuf/runtime/protoimpl
	$(GOGET) -u google.golang.org/grpc
	$(GOGET) -u google.golang.org/grpc/codes
	$(GOGET) -u google.golang.org/grpc/status
	$(GOBUILD) -o $(BINARY_NAME) -v