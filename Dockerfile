FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /geoduel .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /geoduel ./geoduel
COPY web ./web
EXPOSE 8080
CMD ["./geoduel"]
