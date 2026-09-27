FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download && go build -o auth .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /app/auth .
EXPOSE 8080
ENTRYPOINT ["./auth"]
