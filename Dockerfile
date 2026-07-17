ARG GO_VERSION=1.24.6
FROM golang:${GO_VERSION}-alpine3.22 AS build

ARG BINARY=collector
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/service "./cmd/${BINARY}"

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=build /out/service /app/service
USER app
ENTRYPOINT ["/app/service"]
