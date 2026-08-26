# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/meme-cli .

FROM alpine:3.20
RUN adduser -D -u 1000 meme
COPY --from=build /out/meme-cli /usr/local/bin/meme-cli
USER meme
WORKDIR /home/meme

# MEME_DIR is unset by default, so the binary falls back to its embedded
# seed template library. Point it at a volume mount to bring your own
# templates instead, e.g.:
#   docker run -v ./my-templates:/templates -e MEME_DIR=/templates meme-cli list
ENTRYPOINT ["meme-cli"]
CMD ["--help"]
