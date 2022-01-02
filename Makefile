.PHONY: all clean

all: cli node

cli:
	cd cmd/cli && \
	go build -o ../../bin/cli && \
	cd ../..

node:
	cd cmd/node && \
	go build -o ../../bin/node && \
	cd ../..

clean:
	rm bin/*