.PHONY: server

FILES  = $(shell git ls-files)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD | awk -F'/' '{print $$NF}')

IMGNAME = cast
TAGNAME = $(BRANCH)

DC_PROD = docker-compose -f tools/docker-compose.prod.yml
DC_DEV  = docker-compose -f tools/docker-compose.dev.yml

default: all

all: install
deps: serverDeps kodaDeps workerDeps

gvt:
	go get -u github.com/FiloSottile/gvt
	gvt restore

server:
	@cd src/github.com/cjlucas/unnamedcast/server; go install
worker:
	@cd src/github.com/cjlucas/unnamedcast/worker; go install

install: server worker

localTest:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test

buildContext:
	rm -rf build
	mkdir build
	@echo "Copying project to /build..."
	@git ls-files | cpio -pdm build/ 2> /dev/null

devBuild: buildContext
	$(DC_DEV) build web
	$(DC_DEV) build worker
	$(DC_DEV) build watcher

prodBuild: buildContext
	$(DC_PROD) build web
	$(DC_PROD) build worker

test:
	$(DC_DEV) build web
	@${DC_DEV} run -e DB_URL=mongodb://db/casttest web make localTest

deploy: prodBuild
	$(DC_PROD) up

watch: devBuild
	@$(DC_DEV) up

docker: buildContext
	@echo "Building docker image..."
	@cd build; docker build -f tools/Dockerfile -t $(IMGNAME):$(TAGNAME) .
	@echo "Run image: docker run -it $(IMGNAME):$(TAGNAME)"
