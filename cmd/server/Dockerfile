FROM golang:1.13.4 as builder

RUN mkdir /app 
WORKDIR /app 
COPY go.mod go.sum ./
RUN go mod download
COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/server/main.go

FROM alpine
COPY --from=builder /app/main /app/main
COPY --from=builder /app/api/server/ip2location-ip4.bin /app
WORKDIR /app 
ENTRYPOINT ["./main"]