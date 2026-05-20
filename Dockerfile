FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/max-bot ./cmd/bot

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/max-bot /app/max-bot
EXPOSE 8080
ENTRYPOINT ["/app/max-bot"]
