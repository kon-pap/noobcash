.PHONY: all clean cli node
all: node cli

CLI_DEPS:= $(wildcard ./pkg/cli/**.go)
NODE_DEPS:= $(wildcard ./pkg/node/*/**.go ./pkg/node/*.go ./pkg/env/*.go)

node: bin/node
bin/node: ./cmd/node/main.go $(NODE_DEPS)
	cd cmd/node && go build -o ../../bin/node && cd ../..

cli: bin/cli
bin/cli: ./cmd/cli/main.go $(CLI_DEPS)
	cd cmd/cli && go build -o ../../bin/cli && cd ../..

clean:
	rm -r bin/*