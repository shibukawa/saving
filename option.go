package saving

import (
	"log/slog"
	"net/url"
	"time"
)

type PortMap struct {
	FromPort    string
	Destination *url.URL
}

type Option struct {
	HealthCheckUrl     *url.URL      // Health check URL
	WakeTimeout        time.Duration // Timeout duration to wait before scaling up the backend server
	DrainTimeout       time.Duration // Timeout duration to wait before scaling down the backend server
	HealthCheckTimeout time.Duration // Timeout duration to wait oneshot health check request
	PortToDest         []PortMap     // map of listening port to destination
	Logger             *slog.Logger  // Logger
	Cmd                string        // Command to execute
	Args               []string      // Command args
	PidPath            string        // Pid file that stores the process ID
}

type ProcessOption struct {
	PidPath            string
	HealthCheckUrl     *url.URL
	WakeTimeout        time.Duration
	DrainTimeout       time.Duration
	HealthCheckTimeout time.Duration
	Cmd                string
	Args               []string
	Logger             *slog.Logger
}

func (o Option) ToProcessOption() ProcessOption {
	return ProcessOption{
		PidPath:        o.PidPath,
		HealthCheckUrl: o.HealthCheckUrl,
		WakeTimeout:    o.WakeTimeout,
		DrainTimeout:   o.DrainTimeout,
		Cmd:            o.Cmd,
		Args:           o.Args,
		Logger:         o.Logger,
	}
}
