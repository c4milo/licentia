test:
	go test ./...

install:
	go install

build:
	go build

licenses:
	go-bindata -o=licenses.go licenses

.PHONY: build licenses install test

