
# saving: command wrapper puts server process to sleep until request is coming

It reduces memory usage of container until request is coming without sidecar container. This idea is similar to xinetd or systemd socket activation (Accept=no), but it works as a command wrapper.

## FAQ

* Is it good for cloud platform use?
  - No. Google Cloud's Cloud Run is already have a scale to zero feature. Any other container service uses fixed resources even if the server process is not running.
* Is it good for development use?
  - Yes. It reduces memory of local container engines. If you 
* Is it good for servers written in any programming language?
  - No. It launches server process when request is coming. If the server process takes too long to start, it is difficult to use.
* Is it good for databases?
  - No. It supports only HTTP requests.

## Usage and Mechanism

It assumed to use with the following Dockerfile definition:

```Dockerfile
# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.3

# Download prebuild saving command
FROM --platform=$BUILDPLATFORM ghcr.io/shibukawa/saving:latest AS install-saving

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
FROM gcr.io/distroless/base-debian12 AS final

COPY --from=install-saving /bin/saving /bin/saving
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
```

`saving` command is works as a `ENTRYPOINT` like shell. And also works as a health checker. Your server process is passed as `CMD` argument. You can copy prebuild binary from `ghcr.io/shibukawa/saving:latest` image or build it from source (See [Dockerfile](https://github.com/shibukawa/saving/blob/main/Dockerfile)).

`saving` command is a reverse proxy. It waits for incoming requests and starts your server process when request is coming. It waits for the server process to finish and then goes to sleep again.

Options are passed via environment variables. 

## Option and Environment Variables

```bash
$ sleepable [OPTIONs] -- command ...
```

It accepts options:

* `-h`, `--help`: Show help message and exit.
* `-verbose`: Show more logs to stderr (it is as same as `SAVING_SLOG_LOG_LEVEL=info`).
* `-health-check`: Run health check and exit.

It accepts environment variables to configure its behavior:

* `SAVING_PORT_MAPS`: Port mappings in the format `waiting_port:server_port`. It is required.
* `SAVING_DRAIN_TIMEOUT`: Time to wait for the server process to finish before stopping it (default: `1m`).
* `SAVING_WAKE_TIMEOUT`: Time to wait for the server process to start before giving up (default: `10s`).
* `SAVING_HEALTH_CHECK_PORT`: Port to use for health checks (default: `8080`).
* `SAVING_HEALTH_CHECK_PATH`: Path to the health check endpoint (default: `/health`).
* `SAVING_PID_PATH`: Path to the file where the PID of the server process is stored (default: `/$TMP/SAVING_PID`).

It has additional options for logging configuration:

* `SAVING_SLOG_FORMAT`: Log format, can be `json` or `text` (default: `text`).
* `SAVING_SLOG_LOG_LEVEL`: Log level, can be `debug`, `info`, `warning`, `error` (default: `warning`).
* `SAVING_SLOG_ADD_SOURCE`: Whether to add source information to logs, can be `yes` or `no` (default: `no`).
* `SAVING_SLOG_LOG_EXTRA`: Additional log fields in `key1=value1,key2=value2` format (default: `''`).

## License

AGPL-3.0
