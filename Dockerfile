# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.3

# Install saving command
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build-saving
WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -trimpath -o /bin/saving ./cmd/saving

# Build sample target web service
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build-sample
WORKDIR /work
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=./example/go.mod,target=go.mod \
    go mod download -x
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=./example/go.mod,target=go.mod \
    --mount=type=bind,source=./example/server.go,target=server.go \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/sample-server .

# Deploy image
FROM gcr.io/distroless/base-debian12:debug AS final

COPY --from=build-saving /bin/saving /bin/saving
COPY --from=build-sample /bin/sample-server /bin/sample-server

EXPOSE 80

ENV SAVING_DRAIN_TIMEOUT=10s
ENV SAVING_WAKE_TIMEOUT=10s
ENV SAVING_HEALTH_CHECK_PATH=/health
ENV SAVING_PORT_MAPS=80:8080

ENV SAVING_SLOG_LOG_LEVEL=info
ENV SAVING_SLOG_FORMAT=json
ENV SAVING_SLOG_ADD_SOURCE=no

ENTRYPOINT [ "/bin/saving" ]
HEALTHCHECK CMD ["/bin/saving", "-health-check"]
CMD [ "/bin/sample-server" ]
