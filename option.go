package saving

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultPidFilename = "SAVING_PID"

type PortMap struct {
	FromPort    string
	Destination *url.URL
}

type Option struct {
	HealthCheckUrl     *url.URL      // Health check URL
	WakeTimeout        time.Duration // Timeout duration to wait before scaling up the backend server
	DrainTimeout       time.Duration // Timeout duration to wait before scaling down the backend server
	HealthCheckTimeout time.Duration // Timeout duration to wait oneshot health check request
	PortMaps           []PortMap     // map of listening port to destination
	Logger             *slog.Logger  // Logger
	Cmd                string        // Command to execute
	Args               []string      // Command args
	PidPath            string        // Pid file that stores the process ID
}

var ErrParseOption = errors.New("parse option error")

func InitOption(args []string) (*Option, error) {
	result := &Option{
		PidPath: NormalizePidPath(os.Getenv("SAVING_PID_PATH")),
	}
	if len(args) > 0 {
		result.Cmd = args[0]
		result.Args = args[1:]
	}
	var errs []error
	if drainTimeout, valid := NormalizeDuration(os.Getenv("SAVING_DRAIN_TIMEOUT"), 1*time.Minute); !valid {
		errs = append(errs, fmt.Errorf("%w: SAVING_DRAIN_TIMEOUT is invalid: '%s'", ErrParseOption, os.Getenv("SAVING_DRAIN_TIMEOUT")))

	} else {
		result.DrainTimeout = drainTimeout
	}
	if wakeTimeout, valid := NormalizeDuration(os.Getenv("SAVING_WAKE_TIMEOUT"), 10*time.Second); !valid {
		errs = append(errs, fmt.Errorf("%w: SAVING_WAKE_TIMEOUT is invalid: '%s'", ErrParseOption, os.Getenv("SAVING_WAKE_TIMEOUT")))

	} else {
		result.WakeTimeout = wakeTimeout
	}
	portMaps := strings.Split(os.Getenv("SAVING_PORT_MAPS"), ",")
	result.PortMaps = make([]PortMap, 0, len(portMaps))
	for _, portMap := range portMaps {
		if strings.TrimSpace(portMap) == "" {
			continue
		}
		ports := strings.Split(portMap, ":")
		if len(ports) != 2 {
			errs = append(errs, fmt.Errorf("%w: SAVING_PORT_MAPS: format error: '%s'", ErrParseOption, portMap))
			continue
		} else {
			var listenPort, targetPort uint16
			lp, err := strconv.ParseUint(ports[0], 10, 16)
			if err != nil || lp < 1 || lp > 65535 {
				errs = append(errs, fmt.Errorf("%w: SAVING_PORT_MAPS: listen port should be 1-65535: '%d'", ErrParseOption, lp))
			} else {
				listenPort = uint16(lp)
			}
			tp, err := strconv.ParseUint(ports[1], 10, 16)
			if err != nil || tp < 1 || tp > 65535 {
				errs = append(errs, fmt.Errorf("%w: SAVING_PORT_MAPS: target port should be 1-65535: '%d'", ErrParseOption, tp))
			} else {
				targetPort = uint16(tp)
			}
			if listenPort != 0 && targetPort != 0 {
				u, _ := url.Parse("http://localhost:" + strconv.Itoa(int(targetPort)))
				result.PortMaps = append(result.PortMaps, PortMap{":" + strconv.Itoa(int(listenPort)), u})
			}
		}
	}
	if len(result.PortMaps) == 0 {
		errs = append(errs, fmt.Errorf("%w: SAVING_PORT_MAPS env var is required, but empty", ErrParseOption))
	}
	healthCheckUrl := &url.URL{
		Scheme: "http",
		Path:   os.Getenv("SAVING_HEALTH_CHECK_PATH"),
	}
	if healthCheckUrl.Path == "" {
		healthCheckUrl.Path = "/health"
	}
	healthCheckPort := os.Getenv("SAVING_HEALTH_CHECK_PORT")
	if healthCheckPort == "" {
		if len(result.PortMaps) > 0 {
			healthCheckUrl.Host = result.PortMaps[0].Destination.Host
		}
	} else if p, err := strconv.ParseUint(healthCheckPort, 10, 16); err != nil || p == 0 {
		errs = append(errs, fmt.Errorf("%w: SAVING_HEALTH_CHECK_PORT: port should be 1-65535: '%s'", ErrParseOption, healthCheckPort))
	} else {
		healthCheckUrl.Host = net.JoinHostPort("localhost", healthCheckPort)
	}
	result.HealthCheckUrl = healthCheckUrl
	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}

func NormalizePidPath(pidPath string) string {
	if pidPath == "" {
		return filepath.Join(os.TempDir(), DefaultPidFilename)
	}
	return pidPath
}

func NormalizeDuration(src string, defaultValue time.Duration) (time.Duration, bool) {
	if src == "" {
		return defaultValue, true
	}
	result, err := time.ParseDuration(src)
	if err != nil {
		return 0, false
	}
	return result, true
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
