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
	rm -rf pkg bin
	rm -rf src/github.com/cjlucas/unnamedcast/*/vendor/*/

test:
	@cd src/github.com/cjlucas/unnamedcast; go list ./... | grep -v vendor | xargs go test
