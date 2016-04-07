FILES = $(shell git ls-files)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
IMGNAME = cast
TAGNAME = $(BRANCH)

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

localUnittest:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test

localTest:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs -i go test {} -integration

unittest: docker
	@docker run $(IMGNAME) make localUnitTest

test: dockerCompose
	@docker-compose -f tools/docker-compose.yml run app make localTest

buildContext: clean
	mkdir build
	@echo "Copying project to /build..."
	@$(foreach f, $(FILES), mkdir -p build/$(shell dirname $(f)); cp $(f) build/$(shell dirname $(f));)

dockerCompose: buildContext
	@echo "Building docker image (docker-compose)..."
	@docker-compose -f tools/docker-compose.yml build app

docker: buildContext
	@echo "Building docker image..."
	@cd build; docker build -f tools/Dockerfile -t $(IMGNAME):$(TAGNAME) .
	@echo "Run image: docker run -it $(IMGNAME):$(TAGNAME)"
