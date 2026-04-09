PKG_NAME    := github.com/catalin4513/terraform-provider-uptrace-ce
BINARY      := terraform-provider-uptrace-ce
VERSION     ?= dev
LDFLAGS     := -X $(PKG_NAME)/version.ProviderVersion=$(VERSION)

GO          ?= go
GOFMT_FILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: default
default: build

.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

.PHONY: test
test:
	$(GO) test ./... -count=1

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	gofmt -s -w $(GOFMT_FILES)

.PHONY: fmtcheck
fmtcheck:
	@unformatted=$$(gofmt -s -l $(GOFMT_FILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt'd:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: clean
clean:
	rm -f $(BINARY)
