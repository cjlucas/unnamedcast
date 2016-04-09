FILES = $(shell git ls-files)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
IMGNAME = cast
TAGNAME = $(BRANCH)

default: all

all: server koda worker
deps: serverDeps kodaDeps workerDeps

fix:
	@cd src/github.com/cjlucas/unnamedcast/server; go get -fix
	@cd src/github.com/cjlucas/unnamedcast/worker; go get -fix
	@cd src/github.com/cjlucas/unnamedcast/koda; go get -fix

install:
	@cd src/github.com/cjlucas/unnamedcast/server; go install
	@cd src/github.com/cjlucas/unnamedcast/worker; go install

gvt:
	go get -u github.com/FiloSottile/gvt
	gvt restore

serverDeps: gvt
	cd src/github.com/cjlucas/unnamedcast/server; gvt restore

kodaDeps: gvt
	cd src/github.com/cjlucas/unnamedcast/koda; gvt restore

workerDeps: gvt
	cd src/github.com/cjlucas/unnamedcast/worker; gvt restore

server: serverDeps
koda: kodaDeps
worker: workerDeps

localUnittest:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test

# TODO: figure out a good method for executing integration tests
localTest: localUnittest

unittest: docker
	@docker run $(IMGNAME):$(TAGNAME) make localUnittest

test: dockerCompose
	@docker-compose -f tools/docker-compose.yml run app make localTest

buildContext:
	rm -rf build
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
