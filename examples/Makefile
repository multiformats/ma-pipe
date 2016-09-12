
all: deps

deps: ipfs ma-pipe

ipfs:
	@which ipfs >/dev/null && ln -s $(shell which ipfs) && echo "using $(shell which ipfs)"
	@which ipfs >/dev/null || echo "please install ipfs in this directory"

ma-pipe:
	@echo "building ma-pipe"
	go build -o ma-pipe ../ma-pipe

clean:
	rm ma-pipe
	rm -rf tmp
