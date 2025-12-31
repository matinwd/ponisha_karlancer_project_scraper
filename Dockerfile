FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /bin/server /app/server
COPY db/schema.sql /app/db/schema.sql

ENV HTTP_PORT=3000
EXPOSE 3000

CMD ["/app/server"]
