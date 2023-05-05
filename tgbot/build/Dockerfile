FROM golang:latest AS build

RUN apt-get update && apt-get install -y git

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o ./build/main ./cmd/tgbot/

FROM mongo:latest

COPY --from=build /app/build/main /usr/local/bin/main

WORKDIR /usr/local/bin

EXPOSE 8080

CMD ["main"]
