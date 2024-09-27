
all:	generate build test

generate:
	go generate

build:
	go build

test:
	go test -v ./...

run:
	sudo ./ebpf-test

clean:
	rm -f *.o
	rm -f counter_bpfeb.go
	rm -f ebpf-test

