GOFMT_FILES?=$$(find . -name '*.go')
TEST?=$$(go list ./...)
export GO111MODULE=on
export TF_ACC_TERRAFORM_VERSION=0.15.4
export TESTARGS=-race -coverprofile=coverage.txt -covermode=atomic

default: build

build:
	go install

dist:
	goreleaser build --single-target --skip-validate --clean

test:
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc:
	TF_ACC=1 go test ./internal/provider -v $(TESTARGS) -timeout 120m -count=1

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@bash scripts/gofmtcheck.sh

vet:
	@echo "go vet ."
	@go vet $$(go list ./...) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi
