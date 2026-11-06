FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth .

FROM alpine:3.20

RUN adduser -D -H appuser
COPY --from=build /out/auth /usr/local/bin/auth
USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/auth"]
