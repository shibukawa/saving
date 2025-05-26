package sloginit

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

var ErrInitSlog = errors.New("init slog error")

type LogType int

const (
	JsonLog LogType = iota + 1
	TextLog
)

// InitSlog initialize logger by using environment variables
//
// If the process name is "program", it refers the following variables:
//   - PROGRAM_SLOG_FORMAT: text(default) or json
//   - PROGRAM_SLOG_ADD_SOURCE: 0/no/off/false(default) or others
//   - PROGRAM_SLOG_LOG_LEVEL: debug/info/warn(default)/error
//   - PROGRAM_SLOG_LOG_EXTRA: key1=value1,key2=value2
func InitSlog(program string, w io.Writer, verbose bool) (*slog.Logger, LogType, error) {
	prefix := strings.ToUpper(program)
	if w == nil {
		w = os.Stderr
	}

	var opt slog.HandlerOptions
	ss, have := os.LookupEnv(prefix + "_SLOG_ADD_SOURCE")
	if !have || ss == "0" || ss == "off" || ss == "false" || ss == "no" {
		opt.AddSource = false
	} else {
		opt.AddSource = true
	}

	var level slog.Level
	switch os.Getenv(prefix + "_SLOG_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		fallthrough
	case "warning":
		fallthrough
	case "":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, 0, fmt.Errorf("%w: wrong error level: %s", ErrInitSlog, os.Getenv(prefix+"_SLOG_LOG_LEVEL"))
	}
	if verbose && level > slog.LevelInfo {
		level = slog.LevelInfo
	}
	opt.Level = &leveler{level}

	var h slog.Handler
	var lt LogType
	switch os.Getenv(prefix + "_SLOG_FORMAT") {
	case "text":
		fallthrough
	case "":
		h = slog.NewTextHandler(w, &opt)
		lt = TextLog
	case "json":
		h = slog.NewJSONHandler(w, &opt)
		lt = JsonLog
	default:
		return nil, 0, fmt.Errorf("%w: wrong format: %s", ErrInitSlog, os.Getenv(prefix+"_SLOG_FORMAT"))
	}
	result := slog.New(h).With("program", program)

	for _, e := range strings.Split(os.Getenv(prefix+"_SLOG_LOG_LEVEL"), ",") {
		tokens := strings.SplitN(e, "=", 2)
		if len(tokens) == 2 {
			result = result.With(strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1]))
		}
	}

	return result, lt, nil
}

type leveler struct {
	l slog.Level
}

func (l leveler) Level() slog.Level {
	return l.l
}

var _ slog.Leveler = (*leveler)(nil)
