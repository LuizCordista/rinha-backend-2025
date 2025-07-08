FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
#COPY go.sum ./
RUN go mod download

COPY . ./

WORKDIR /app/cmd/api
RUN go build -o /hello-app

# Final image
FROM alpine:latest
COPY --from=builder /hello-app /hello-app

EXPOSE 8080

CMD ["/hello-app"]
