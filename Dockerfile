FROM golang:1.25 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o social-media-scaling .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/social-media-scaling .

EXPOSE 8080
ENTRYPOINT ["./social-media-scaling"]
