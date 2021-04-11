FROM golang:1.16-alpine AS build

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
WORKDIR /tmp/crawler

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN  go build -trimpath -ldflags "-s -w -extldflags '-static'" -o /bin/crawler ./cmd/crawler/main.go

FROM alpine:3.13
COPY --from=build /bin/crawler /bin/crawler
EXPOSE 8080
CMD ["/bin/crawler"]
