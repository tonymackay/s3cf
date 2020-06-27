FROM golang:alpine AS s3cf-builder
ARG GIT_TAG=dev
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN go mod download
RUN go build -o s3cf -ldflags=-X=main.version=${GIT_TAG}

FROM amazon/aws-cli
COPY --from=s3cf-builder /build/s3cf /usr/local/bin/s3cf
WORKDIR /s3cf
ENTRYPOINT ["/usr/local/bin/s3cf"]
