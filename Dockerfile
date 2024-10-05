FROM golang:1.23.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o scraper ./cmd/scraper/main.go

ENTRYPOINT ["./scraper"]
CMD ["1", "2"]
