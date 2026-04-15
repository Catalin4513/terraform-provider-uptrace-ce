PKG_NAME    := github.com/catalin4513/terraform-provider-uptrace-ce
BINARY      := terraform-provider-uptrace-ce
VERSION     ?= dev
LDFLAGS     := -X $(PKG_NAME)/version.ProviderVersion=$(VERSION)

GO          ?= go
GOFMT_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './internal/generated/*')

.PHONY: default
default: build

.PHONY: generate
generate:
	rm -f internal/generated/*.go
	$(GO) run github.com/uptrace/oapi-codegen-dd/v3/cmd/oapi-codegen \
		-config oapi-codegen.yaml openapi/openapi.yaml

.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

.PHONY: test
test:
	$(GO) test ./... -count=1

.PHONY: testacc
testacc:
	TF_ACC=1 $(GO) test ./... -count=1 -timeout 10m

VET_PKGS := $(shell $(GO) list ./... | grep -v /internal/generated)

.PHONY: vet
vet:
	$(GO) vet $(VET_PKGS)

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
