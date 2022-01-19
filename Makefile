.PHONY: all clean cli node
all: node cli

# https://stackoverflow.com/a/6145814/13537527
FILTER_OUT = $(foreach v,$(2),$(if $(findstring $(1),$(v)),,$(v))) 

CLI_DEPS:= $(wildcard ./pkg/cli-utils/cli/*.go )
NODE_DEPS:= $(wildcard ./pkg/node/backend/*.go ./pkg/node/*.go ./pkg/env/*.go ./pkg/cli-utils/node/*.go)

 # filter out test files
CLI_DEPS:= $(call FILTER_OUT,_test.go, $(CLI_DEPS))
NODE_DEPS:= $(call FILTER_OUT,_test.go, $(NODE_DEPS))

node: bin/noobcash-node
bin/noobcash-node: ./cmd/node/main.go $(NODE_DEPS)
	cd cmd/node && go build -o ../../bin/noobcash-node && cd ../..

cli: bin/noobcash-cli
bin/noobcash-cli: ./cmd/cli/main.go $(CLI_DEPS)
	cd cmd/cli && go build -o ../../bin/noobcash-cli && cd ../..

test:
	go test ./... -v || echo -n ""

clean:
	rm -r bin/*