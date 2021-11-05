.PHONY: all clean test swp swampd

BINNAME=swamp

all: ${BINNAME}

release: swampd swamp swp
	upx --force swamp
	upx --force swampd
	upx --force swp

${BINNAME}: swampd swp
	./script/compile-resources
	go build -ldflags="-s -w -X 'github.com/swampapp/swamp/internal/version.GIT_SHA=$(shell git rev-parse --short HEAD)'" -o ${BINNAME}

swampd:
	go build -ldflags="-s -w -X 'github.com/swampapp/swamp/internal/version.GIT_SHA=$(shell git rev-parse --short HEAD)'" -o swampd ./cmd/swampd

swp:
	go build -ldflags="-s -w -X 'github.com/swampapp/swamp/internal/version.GIT_SHA=$(shell git rev-parse --short HEAD)'" -o swp ./cmd/swp

clean:
	rm -f ${BINNAME} swampd swp

test: ${BINNAME}
	./script/test
