FROM golang:1.17
WORKDIR /go/src/github.com/streamingfast/substreams-playground/
COPY . ./
RUN go get ./...
RUN GOOS=linux go build -o sseth ./cmd/sseth
RUN mkdir /app/ && mv ./sseth /app/
WORKDIR /app/