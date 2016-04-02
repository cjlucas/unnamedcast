default: all

all: server koda worker

server: gvt
	cd src/github.com/cjlucas/unnamedcast/server; gvt restore

koda: gvt
	cd src/github.com/cjlucas/unnamedcast/koda; gvt restore

worker: gvt
	cd src/github.com/cjlucas/unnamedcast/worker; gvt restore

gvt:
	go get -u github.com/FiloSottile/gvt
	gvt restore

clean:
	rm -rf pkg bin
	rm -rf src/github.com/cjlucas/unnamedcast/*/vendor/*/
