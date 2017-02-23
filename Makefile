NAME := licentia
VERSION := v1.0.1

build:
	go build -ldflags "-X main.Version=$(VERSION)"

compile:
	@rm -rf build/
	@gox -ldflags "-X main.Version=$(VERSION)" \
	-os="darwin" \
	-os="linux" \
	-os="windows" \
	-os="solaris" \
	-output "build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)" \
	./...

install:
	go install -ldflags "-X main.Version=$(VERSION)"

deps:
	go get github.com/c4milo/github-release
	go get github.com/mitchellh/gox

test:
	go test ./...

dist: compile
	$(eval FILES := $(shell ls build))
	@rm -rf dist/
	@mkdir dist/
	@for f in $(FILES); do \
		(cd $(shell pwd)/build/$$f && tar -cvzf ../../dist/$$f.tar.gz *); \
		(cd $(shell pwd)/dist && shasum -a 512 $$f.tar.gz > $$f.sha512); \
		echo $$f; \
	done

release: dist
	@latest_tag=$$(git describe --tags `git rev-list --tags --max-count=1`); \
	comparison="$$latest_tag..HEAD"; \
        if [ -z "$$latest_tag" ]; then comparison=""; fi; \
        changelog=$$(git log $$comparison --oneline --no-merges --reverse); \
        github-release c4milo/$(NAME) $(VERSION) "$$(git rev-parse --abbrev-ref HEAD)" "**Changelog**<br/>$$changelog" 'dist/*'; \
        git pull

licenses:
	go-bindata -o=licenses.go licenses

.PHONY: compile build licenses install test release deps

