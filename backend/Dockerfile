FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.19
RUN adduser -D -u 10001 app
COPY --from=builder /out/api /usr/local/bin/api
USER app
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/api"]
