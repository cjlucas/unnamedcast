FILES = $(shell git ls-files)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD | awk -F'/' '{print $$NF}')
IMGNAME = cast
TAGNAME = $(BRANCH)

default: all

all: server koda worker
deps: serverDeps kodaDeps workerDeps

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
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test -v

# TODO: figure out a good method for executing integration tests
localTest: localUnittest

unittest: docker
	@docker run $(IMGNAME):$(TAGNAME) make localUnittest

test: dockerCompose
	@docker-compose -f tools/docker-compose.yml run web make localTest

buildContext:
	rm -rf build
	mkdir build
	@echo "Copying project to /build..."
	@$(foreach f, $(FILES), mkdir -p build/$(shell dirname $(f)); cp $(f) build/$(shell dirname $(f));)

dockerCompose: buildContext
	@echo "Building docker image (docker-compose)..."
	@docker-compose -f tools/docker-compose.yml build web
	@docker-compose -f tools/docker-compose.yml build worker

docker: buildContext
	@echo "Building docker image..."
	@cd build; docker build -f tools/Dockerfile -t $(IMGNAME):$(TAGNAME) .
	@echo "Run image: docker run -it $(IMGNAME):$(TAGNAME)"
