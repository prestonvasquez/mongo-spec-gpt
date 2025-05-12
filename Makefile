BINARY_NAME=mongo-spec-gpt
CMD_DIR=cmd
OUTPUT_NAME=$(BINARY_NAME)
GOPATH_BIN=$(shell go env GOPATH)/bin

.PHONY: all build install clean 

all: build 

build:
	cd $(CMD_DIR) && go build -o $(OUTPUT_NAME) -v
	$(MAKE) install
	$(MAKE) clean

install: 
	@if [ -f "$(GOPATH_BIN)/$(BINARY_NAME)" ]; then rm -rf $(GOPATH_BIN)/$(BINARY_NAME); fi
	mv $(CMD_DIR)/$(OUTPUT_NAME) $(GOPATH_BIN)/$(BINARY_NAME)

clean:
	rm -f $(CMD_DIR)/$(OUTPUT_NAME)
