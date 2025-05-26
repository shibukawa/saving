package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/shibukawa/saving"
	"github.com/shibukawa/saving/sloginit"
)

var cliOpt struct {
	PidPath         string        `env:"SAVING_PID_PATH" optional:"" default:""`
	DrainTimeout    time.Duration `env:"SAVING_DRAIN_TIMEOUT" optional:"" default:"5m"`
	WakeTimeout     time.Duration `env:"SAVING_WAKE_TIMEOUT" optional:"" default:"10s"`
	HealthCheckPort uint16        `env:"SAVING_HEALTH_CHECK_PORT" optional:""`
	HealthCheckPath string        `env:"SAVING_HEALTH_CHECK_PATH" optional:"" default:"/health"`

	Verbose bool `short:"v" help:"Show information to stderr."`

	Run struct {
		Port []string `flag:"" short:"p" required:"" help:"Port mapping like 80:8000"`
		Cmd  string   `arg:"" name:"cmd" help:"Command to execute"`
		Args []string `arg:"" name:"args" optional:"" help:"Command arguments"`
	} `cmd`

	HealthCheck struct {
	} `cmd`
}

func main() {
	ctx := kong.Parse(&cliOpt)

	logger, logType, err := sloginit.InitSlog("saving", os.Stderr, cliOpt.Verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger config error: %s\n", err.Error())
		os.Exit(1)
	}

	switch ctx.Command() {
	case "run <cmd>":
		hasError := false

		portMaps := make([]saving.PortMap, len(cliOpt.Run.Port))
		for i, portMap := range cliOpt.Run.Port {
			ports := strings.Split(portMap, ":")
			if len(ports) != 2 {
				logger.Error("port mapping error", "input", portMap)
				hasError = true
			} else {
				var listenPort, targetPort uint16
				lp, err := strconv.ParseUint(ports[0], 10, 16)
				if err != nil || lp < 1 || lp > 65535 {
					logger.Error("listen port should be 1-65535", "input", lp)
					hasError = true
				} else {
					listenPort = uint16(lp)
				}
				tp, err := strconv.ParseUint(ports[1], 10, 16)
				if err != nil || tp < 1 || tp > 65535 {
					logger.Error("target port should be 1-65535", "input", lp)
					hasError = true
				} else {
					targetPort = uint16(tp)
				}
				if listenPort != 0 && targetPort != 0 {
					u, _ := url.Parse("http://localhost:" + strconv.Itoa(int(targetPort)))
					portMaps[i] = saving.PortMap{":" + strconv.Itoa(int(listenPort)), u}
				}
			}
		}
		healthCheckUrl := &url.URL{
			Scheme: "http",
			Path:   cliOpt.HealthCheckPath,
		}
		if cliOpt.HealthCheckPort == 0 {
			healthCheckUrl.Host = portMaps[0].Destination.Host
		} else {
			healthCheckUrl.Host = net.JoinHostPort("localhost", strconv.Itoa(int(cliOpt.HealthCheckPort)))
		}

		if hasError {
			os.Exit(1)
		}

		opt := saving.Option{
			HealthCheckUrl: healthCheckUrl,
			WakeTimeout:    cliOpt.WakeTimeout,
			DrainTimeout:   cliOpt.DrainTimeout,
			PortToDest:     portMaps,
			Logger:         logger,
			Cmd:            cliOpt.Run.Cmd,
			Args:           cliOpt.Run.Args,
			PidPath:        saving.NormalizePidPath(cliOpt.PidPath),
		}

		attrs := []any{
			slog.String("cmd", strings.TrimSpace(opt.Cmd+" "+strings.Join(opt.Args, " "))),
			slog.String("health_check_url", opt.HealthCheckUrl.String()),
			slog.Duration("drain_timeout", opt.DrainTimeout),
			slog.Duration("wake_timeout", opt.WakeTimeout),
			slog.String("pid_path", opt.PidPath),
		}
		ports := make([]any, len(portMaps)*2)
		for i, p := range portMaps {
			ports[i*2] = slog.String("from", p.FromPort)
			ports[i*2+1] = slog.String("dest", p.Destination.String())
		}
		if logType == sloginit.JsonLog {
			group := []any{}
			for portMap := range slices.Chunk(ports, 2) {
				group = append(group, slog.Group("port_map", portMap...))
			}
			attrs = append(attrs, portMaps)
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
		err := saving.StartProxy(ctx, opt)
		if err != nil {
			logger.Error("initialization error", "detail", err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	case "healthcheck":
		result := saving.CheckProcessHealth(cliOpt.PidPath)
		logger.Info("health check", "result", result)
		if result {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
