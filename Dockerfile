FROM golang:1.13-alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./cmd/ ./
RUN go build -o main .
EXPOSE 5000

CMD ["./main"]