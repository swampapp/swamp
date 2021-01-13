.PHONY: all clean test swp swampd

BINNAME=swamp

all: ${BINNAME}

${BINNAME}: swampd swp
	./script/compile-resources
	go build -ldflags="-s -w" -o ${BINNAME}
	upx --force swamp

swampd:
	go build -ldflags="-s -w" -o swampd ./cmd/swampd
	upx --force swampd
	cp swampd internal/resources

swp:
	go build -ldflags="-s -w" -o swp ./cmd/swp
	upx --force swp

clean:
	rm -f ${BINNAME} swampd

test: ${BINNAME}
	./script/test
