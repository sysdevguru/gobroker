# -p 6 is to prevent circleci from klling the
# database test due to memory consumption
THREADS?=6
TEST_OPT=-cover -failfast -p $(THREADS)

all:
	go install .

install: all

vendor:
	go mod vendor

update:
	go mod tidy

test: resetdb
	go fmt ./...
	BROKER_MODE=DEV PGDATABASE=gbtest go test ./... $(TEST_OPT)

resetdb:
	psql -U postgres -c 'DROP DATABASE IF EXISTS gbtest'
	psql -U postgres -c 'CREATE DATABASE gbtest'
	PGDATABASE=gbtest go run tools/migrate/migrate.go

login:
	docker login -u alpacamarkets -p crazyTrader9

vendor-in-docker:
	docker run -v $(shell pwd):/go/src/github.com/alpacahq/gobroker -w /go/src/github.com/alpacahq/gobroker --rm alpacamarkets/gobroker:latest go mod vendor

runit-as-sidecar:
	docker run -it \
	-v $(PWD):/go/src/github.com/alpacahq/gobroker \
	-w /go/src/github.com/alpacahq/gobroker \
	--name ib.gobroker.sidecar \
	--net container:ib.net \
	--rm \
	alpacamarkets/gobroker:master bash

build:
	docker pull alpacamarkets/gopaca:latest
	docker build -t alpacamarkets/gobroker:$(DOCKER_TAG) .

push: login build
	docker push alpacamarkets/gobroker:$(DOCKER_TAG)

build-test:
	docker pull alpacamarkets/gopaca:latest
	docker build -t alpacamarkets/gobroker-test:latest integration/

push-test: login build-test
	docker push alpacamarkets/gobroker-test:latest

doc:
	cd docs/apidoc && python -m SimpleHTTPServer
