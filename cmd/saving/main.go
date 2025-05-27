package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"

	"github.com/shibukawa/saving"
	"github.com/shibukawa/saving/sloginit"
)

var (
	helps = []string{
		`SAVING_PORT_MAPS             : (required)It is a port mapping settings like 80:8000. Comma separated.`,
		`SAVING_DRAIN_TIMEOUT         : Timeout duration after last request to scale in (default=1m)`,
		`SAVING_WAKE_TIMEOUT          : Timeout duration when the process is ready after initial request (default=10s)`,
		`SAVING_PID_PATH              : PID file location (default=$TMP/SAVING_PID)`,
		`SAVING_HEALTH_CHECK_PORT     : Health check port (default=initial target port of SAVING_PORT_MAPS)`,
		`SAVING_HEALTH_CHECK_PATH     : Health check path (default=/health)`,
		``,
		`SAVING_SLOG_FORMAT           : Log format. 'text' or 'json' is acceptable (default=text)`,
		`SAVING_SLOG_ADD_SOURCE       : Add source location to log (default=no)`,
		`SAVING_SLOG_LOG_LEVEL        : Log Level. 'debug', 'info', 'warning', 'error' is acceptable (default=warning)`,
		`SAVING_SLOG_LOG_EXTRA        : Additional values to log. 'key1=value1,key2=value2' style config is acceptable`,
	}
)

func main() {
	help := flag.Bool("help", false, "Help")
	verbose := flag.Bool("verbose", false, "Put many logs")
	healthCheck := flag.Bool("health-check", false, "health check")
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "%s [option] [cmd] [args...]\n\nEnvironment Variables:\n", os.Args[0])
		for _, h := range helps {
			fmt.Fprintln(os.Stderr, "  "+h)
		}
		os.Exit(0)
	}

	opt, err := saving.InitOption(flag.Args())
	if err != nil {
		if errs, ok := err.(interface{ Unwrap() []error }); ok {
			fmt.Fprintf(os.Stderr, "logger config error\n")
			for _, err := range errs.Unwrap() {
				fmt.Fprintf(os.Stderr, "  * %s\n", err.Error())
			}

		} else {
			fmt.Fprintf(os.Stderr, "logger config error: %s\n", err.Error())
		}
		os.Exit(1)
	}

	logger, logType, err := sloginit.InitSlog("saving", os.Stderr, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger config error: %s\n", err.Error())
		os.Exit(1)
	}
	opt.Logger = logger

	if *healthCheck {
		result := saving.CheckProcessHealth(opt.PidPath)
		logger.Info("health check", "result", result)
		if result {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	} else if flag.NArg() > 0 {
		attrs := []any{
			slog.String("cmd", strings.TrimSpace(opt.Cmd+" "+strings.Join(opt.Args, " "))),
			slog.String("health_check_url", opt.HealthCheckUrl.String()),
			slog.Duration("drain_timeout", opt.DrainTimeout),
			slog.Duration("wake_timeout", opt.WakeTimeout),
			slog.String("pid_path", opt.PidPath),
		}
		ports := make([]any, len(opt.PortMaps)*2)
		for i, p := range opt.PortMaps {
			ports[i*2] = slog.String("from", p.FromPort)
			ports[i*2+1] = slog.String("dest", p.Destination.String())
		}
		if logType == sloginit.JsonLog {
			group := []any{}
			for portMap := range slices.Chunk(ports, 2) {
				group = append(group, slog.Group("port_map", portMap...))
			}
			attrs = append(attrs, group)
			logger.Info("config", attrs...)
		} else {
			for _, attr := range attrs {
				logger.Info("config", attr)
			}
			for portMap := range slices.Chunk(ports, 2) {
				logger.Info("config", portMap...)
			}
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		err := saving.StartProxy(ctx, *opt)
		if err != nil {
			logger.Error("initialization error", "detail", err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	} else {
		logger.Error("command is required")
		os.Exit(1)
	}
}
