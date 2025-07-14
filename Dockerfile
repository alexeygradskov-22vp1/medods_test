FROM golang:1.24

WORKDIR /app

RUN apt-get update && apt-get install -y postgresql-client
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .

RUN go build -o server ./main.go

RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
