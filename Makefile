all:	build

build:	generate
	go build -o snuffle .

generate:
	( cd src/collector/ebpf && make generate )

test:
	go test -v ./...

clean:
	( cd src/collector/ebpf && make clean )
