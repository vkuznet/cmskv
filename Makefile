VERSION=`git rev-parse --short HEAD`
OS := $(shell uname)
ifeq ($(OS),Darwin)
flags=-ldflags="-s -w -X main.version=${VERSION}"
else
flags=-ldflags="-s -w -X main.version=${VERSION} -extldflags -static"
endif

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o cmskv ${flags}

build_all: build_osx build_linux build

build_osx:
	go clean; rm -rf pkg cmskv; GOOS=darwin CGO_ENABLED=0 go build -o cmskv ${flags}

build_linux:
	go clean; rm -rf pkg cmskv; GOOS=linux CGO_ENABLED=0 go build -o cmskv ${flags}

build_power8:
	go clean; rm -rf pkg cmskv; GOARCH=ppc64le GOOS=linux CGO_ENABLED=0 go build -o cmskv ${flags}

build_arm64:
	go clean; rm -rf pkg cmskv; GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o cmskv ${flags}

build_windows:
	go clean; rm -rf pkg cmskv; GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -o cmskv ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	go test -v -bench=.
