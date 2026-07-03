# syntax=docker/dockerfile:1

FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/dwarpal \
    ./cmd/dwarpal

FROM alpine:3.20

RUN apk add --no-cache git ca-certificates \
    # CI containers run as root over volume-mounted repos owned by the host
    # uid; git 2.35+ refuses such repos ("dubious ownership"). Trusting all
    # directories is the standard pattern for single-purpose CI images.
    && git config --system --add safe.directory '*'

COPY --from=builder /out/dwarpal /usr/local/bin/dwarpal

ENTRYPOINT ["dwarpal"]
