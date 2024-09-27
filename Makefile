all:	build

build:	generate
	go build -o snuffle .

generate:
	( cd collector/ebpf && go generate )

test:
	go test -v ./...

clean:
	( cd collector/ebpf && make clean )
