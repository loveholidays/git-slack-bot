FROM golang:1.24 AS builder

RUN apt-get update && \
    apt-get install -y ca-certificates libssl-dev cpio

# Installing dependencies
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
  && go install github.com/onsi/ginkgo/v2/ginkgo@latest

WORKDIR /src
ADD . /src
ENV DOCKER_RUNNING=true
RUN make check build

FROM scratch

COPY --from=builder /src/bin /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app

ENTRYPOINT ["./git-slack-bot"]