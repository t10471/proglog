export
CONFIG_PATH=${HOME}/.proglog/
GOBIN ?= ${CURDIR}/bin

# jobs
JOBS := $(shell getconf _NPROCESSORS_CONF)

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		test/client-csr.json | cfssljson -bare client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client

	mv *.pem *.csr ${CONFIG_PATH}

$(CONFIG_PATH)/model.conf:
	cp test/model.cfg $(CONFIG_PATH)/model.conf

$(CONFIG_PATH)/policy.csv:
	cp test/policy.csv $(CONFIG_PATH)/policy.csv

.PHONY: test
test:
test: $(CONFIG_PATH)/policy.csv $(CONFIG_PATH)/model.conf
	go test -race ./...

.PHONY: compile
compile:
	protoc protos/v1/*.proto \
		--go_out=./internal/proto \
		--go-grpc_out=./internal/proto \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=./protos

.PHONY: gomod
gomod: ## Install modules.
	go mod tidy -compat=1.19
	go mod edit -fmt go.mod

.PHONY: goinstall
goinstall:
	go install github.com/google/wire/cmd/wire@v0.5.0
	go install github.com/daixiang0/gci@v0.6.3
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.48.0

.PHONY: fmt
fmt:
fmt: gomod
	find . -iname "*.go" | xargs -P ${JOBS} $(GOBIN)/gci write \
			-s Standard -s Default \
			-s "Prefix(github.com/travisjeffery/proglog)"
	$(GOBIN)/golangci-lint run --timeout 2m --config ./.golangci.yml --fix

.PHONY: wire
wire: ## Generate build functions.
wire:
	$(GOBIN)/wire $(shell go list ./internal/... | grep -v test)
	$(MAKE) fmt
TAG ?= 0.0.1

build-docker:
	docker build -t github.com/travisjeffery/proglog:$(TAG) .



