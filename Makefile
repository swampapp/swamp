.PHONY: all clean test swp swampd

BINNAME=swamp

all: ${BINNAME}

${BINNAME}: swampd swp
	./script/compile-resources
	go build -o ${BINNAME} 

swampd:
	go build -o swampd ./cmd/swampd
	cp swampd internal/resources

swp:
	go build -o swp ./cmd/swp

clean:
	rm -f ${BINNAME} swampd

test: ${BINNAME}
	./script/test
