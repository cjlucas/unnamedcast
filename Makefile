FILES = $(shell git ls-files)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
IMGNAME = cast/$(BRANCH)

default: all

all: server koda worker

server: gvt koda
	cd src/github.com/cjlucas/unnamedcast/server; gvt restore
	cd src/github.com/cjlucas/unnamedcast/server; go get -fix

koda: gvt
	cd src/github.com/cjlucas/unnamedcast/koda; gvt restore
	cd src/github.com/cjlucas/unnamedcast/koda; go get -fix

worker: gvt koda
	cd src/github.com/cjlucas/unnamedcast/worker; gvt restore
	cd src/github.com/cjlucas/unnamedcast/worker; go get -fix

gvt:
	go get -u github.com/FiloSottile/gvt
	gvt restore

clean:
	rm -rf pkg bin build
	rm -rf src/github.com/cjlucas/unnamedcast/*/vendor/*/

test_int:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test

test: docker
	@docker run $(IMGNAME) make test_int

docker: clean
	mkdir build
	@echo "Copying project to /build..."
	@$(foreach f, $(FILES), mkdir -p build/$(shell dirname $(f)); cp $(f) build/$(shell dirname $(f));)
	@echo "Building docker image..."
	@cd build; docker build -f tools/Dockerfile -t $(IMGNAME) .
	@echo "Run image: docker run -it $(IMGNAME)"
