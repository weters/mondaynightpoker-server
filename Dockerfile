FROM golang:1.15 AS build
WORKDIR /build
COPY go* ./
RUN go mod download
COPY ./cmd/ ./cmd/
COPY ./internal/ ./internal/
COPY ./pkg/ ./pkg/
ARG version
RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
        -o server \
        -ldflags "-X main.Version=$version" \
        ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=build /build/server /bin/server
COPY ./sql/ ./sql/
COPY ./templates/ ./templates/
ENTRYPOINT [ "/bin/server" ]
