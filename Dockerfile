# builder
FROM gcr.io/alpacahq/gopaca AS builder
 
WORKDIR /go/src/github.com/alpacahq/gobroker
COPY . .

ENV GOFLAGS -mod=vendor
RUN go install . ./tools/migrate/ ./tools/sodloader/ ./tools/assetloader/ ./workers/ ./cmd/sidecar/ ./integration/setup

# container
FROM golang:alpine
 
COPY --from=builder /go/bin/* /bin/

RUN apk update && apk --no-cache add vim git tar curl bash postgresql-client

WORKDIR /go/src/github.com/alpacahq/gobroker
COPY . .

ENTRYPOINT ["/bin/gobroker"]
