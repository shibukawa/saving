
# saving: command wrapper puts server process to sleep until request is coming

It reduces memory usage of container until request is coming without sidecar container. This idea is similar to xinetd or systemd socket activation (Accept=no), but it works as a command wrapper.

## Usage and Mechanism

It assumed to use with the following Dockerfile definition:

```Dockerfile
EXPOSE 80

ENV SAVING_DRAIN_TIMEOUT=10s
ENV SAVING_WAKE_TIMEOUT=10s
ENV SAVING_HEALTH_CHECK_PATH=/health
ENV SAVING_PORT_MAPS=80:8080

ENV SAVING_SLOG_LOG_LEVEL=info
ENV SAVING_SLOG_FORMAT=json
ENV SAVING_SLOG_ADD_SOURCE=no

# Use this command as entrypoint
ENTRYPOINT [ "/bin/saving" ]
HEALTHCHECK CMD ["/bin/saving", "-health-check"]
# Use server process as command
CMD [ "/bin/sample-server" ]
```

`saving` command is works as a `ENTRYPOINT` like shell. And also works as a health checker. Your server process is passed as `CMD` argument.

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
