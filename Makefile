# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)


# Used to populate version variable in main package.
VERSION=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)

# Allow turning off function inlining and variable registerization
ifeq (${DISABLE_OPTIMIZATION},true)
	GO_GCFLAGS=-gcflags "-N -l"
	VERSION:="$(VERSION)-noopt"
endif

GO_LDFLAGS=-ldflags "-X `go list ./version`.Version=$(VERSION)"

.PHONY: all build binaries clean dep-restore dep-save dep-validate fmt lint test test-full vet
.DEFAULT: all
all: fmt vet lint build test binaries

AUTHORS: .mailmap .git/HEAD
	 git log --format='%aN <%aE>' | sort -fu > $@

# This only needs to be generated by hand when cutting full releases.
version/version.go:
	./version/version.sh > $@

# Required for go 1.5 to build
GO15VENDOREXPERIMENT := 1

# Go files
GOFILES=$(shell find . -type f -name '*.go')

# Package list
PKGS=$(shell go list -tags "${DOCKER_BUILDTAGS}" ./... | grep -v ^github.com/docker/distribution/vendor/)

# Resolving binary dependencies for specific targets
GOLINT=$(shell which golint || echo '')
VNDR=$(shell which vndr || echo '')

${PREFIX}/bin/registry: $(GOFILES)
	@echo "+ $@"
	@go build -tags "${DOCKER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/registry

${PREFIX}/bin/digest:  $(GOFILES)
	@echo "+ $@"
	@go build -tags "${DOCKER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/digest

${PREFIX}/bin/registry-api-descriptor-template: $(GOFILES)
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ${GO_GCFLAGS} ./cmd/registry-api-descriptor-template

docs/spec/api.md: docs/spec/api.md.tmpl ${PREFIX}/bin/registry-api-descriptor-template
	./bin/registry-api-descriptor-template $< > $@

vet:
	@echo "+ $@"
	@go vet -tags "${DOCKER_BUILDTAGS}" $(PKGS)

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . 2>&1 | grep -v ^vendor/ | tee /dev/stderr)" || \
		(echo >&2 "+ please format Go code with 'gofmt -s'" && false)

lint:
	@echo "+ $@"
	$(if $(GOLINT), , \
		$(error Please install golint: `go get -u github.com/golang/lint/golint`))
	@test -z "$$($(GOLINT) ./... 2>&1 | grep -v ^vendor/ | tee /dev/stderr)"

build:
	@echo "+ $@"
	@go build -tags "${DOCKER_BUILDTAGS}" -v ${GO_LDFLAGS} $(PKGS)

#test:
#	@echo "+ $@"
#	@go test -test.short -tags "${DOCKER_BUILDTAGS}" $(PKGS)

test-full:
	@echo "+ $@"
	@go test -tags "${DOCKER_BUILDTAGS}" $(PKGS)

binaries: ${PREFIX}/bin/registry ${PREFIX}/bin/digest ${PREFIX}/bin/registry-api-descriptor-template
	@echo "+ $@"

clean:
	@echo "+ $@"
	@rm -rf "${PREFIX}/bin/registry" "${PREFIX}/bin/digest" "${PREFIX}/bin/registry-api-descriptor-template"

dep-validate:
	@echo "+ $@"
	$(if $(VNDR), , \
		$(error Please install vndr: go get github.com/lk4d4/vndr))
	@rm -Rf .vendor.bak
	@mv vendor .vendor.bak
	@$(VNDR)
	@test -z "$$(diff -r vendor .vendor.bak 2>&1 | tee /dev/stderr)" || \
		(echo >&2 "+ inconsistent dependencies! what you have in vendor.conf does not match with what you have in vendor" && false)
	@rm -Rf vendor
	@mv .vendor.bak vendor
