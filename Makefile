.PHONY: install

TOP_PACKAGE_DIR := github.com/jzvelc/signaller
GO_LDFLAGS := $$GO_LDFLAGS -X main.Version=$$(cat VERSION) -X $(TOP_PACKAGE_DIR)/lib/helper.gopath=$$GOROOT:$$GOPATH

install:
	@go build -i -ldflags="$(GO_LDFLAGS)" -o $$GOROOT/bin/signaller github.com/jzvelc/signaller

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o release/linux/amd64/signaller github.com/jzvelc/signaller

build-alpine:
	@docker run --rm -it -v $$(pwd):/go/src/github.com/jzvelc/signaller -e "GOOS=linux" -e "GOARCH=amd64" jzvelc/signaller:builder go build -ldflags "$(GO_LDFLAGS)" -gcflags=-trimpath=$${GOPATH} -asmflags=-trimpath=$${GOPATH} -o release/linux/musl/signaller github.com/jzvelc/signaller

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -gcflags=-trimpath=$${GOPATH} -asmflags=-trimpath=$${GOPATH} -o release/darwin/amd64/signaller github.com/jzvelc/signaller

compress-linux:
	@docker run --rm -it -v $$(pwd):/go/src/github.com/jzvelc/signaller -e "GOOS=linux" -e "GOARCH=amd64" jzvelc/signaller:builder upx -7 /go/src/github.com/jzvelc/signaller/release/linux/amd64/signaller || true

compress-alpine:
	@docker run --rm -it -v $$(pwd):/go/src/github.com/jzvelc/signaller -e "GOOS=linux" -e "GOARCH=amd64" jzvelc/signaller:builder upx -7 /go/src/github.com/jzvelc/signaller/release/linux/musl/signaller || true

compress-darwin:
	@docker run --rm -it -v $$(pwd):/go/src/github.com/jzvelc/signaller -e "GOOS=darwin" -e "GOARCH=amd64" jzvelc/signaller:builder upx -7 /go/src/github.com/jzvelc/signaller/release/linux/darwin/amd64/signaller || true

build: build-linux build-alpine build-darwin compress-linux compress-alpine compress-darwin

