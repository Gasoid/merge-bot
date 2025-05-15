FROM golang:1.24.3-bookworm AS builder
WORKDIR /code
ADD go.mod /code/
ADD go.sum /code/
RUN go mod download
ADD ./ /code/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /tmp/bot .

FROM alpine:3.6
EXPOSE 8080 443
WORKDIR /root/
RUN apk --no-cache --update add bash curl less jq openssl git
COPY --from=builder /tmp/bot /root/
# HEALTHCHECK --interval=10s --timeout=3s \
#   CMD curl -f http://localhost:8080/health || exit 1
CMD exec /root/bot
