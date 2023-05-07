FROM golang:1.20 AS goimage

RUN apt-get update && apt-get install -y git

WORKDIR /app

COPY . .

WORKDIR /app/cmd/tgbot
RUN go mod download
RUN go build -o /app/build/main

FROM mongo:6.0.5

COPY --from=goimage /app/build/main /usr/local/bin/main

EXPOSE 8080

CMD ["main"]
