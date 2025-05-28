package saving

import (
	"errors"
	"net/url"
	"os"
	"strconv"
)

var ErrHealthCheckFailed = errors.New("health check failed")

type ProcessController interface {
	Exec(callback func()) error
	IsWaking() bool
	Pid() int
}

func writePid(pidPath string, healthCheckUrl *url.URL) error {
	os.Remove(pidPath)
	f, err := os.Create(pidPath)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(strconv.Itoa(os.Getpid()))
	if healthCheckUrl != nil {
		f.WriteString(":")
		f.WriteString(healthCheckUrl.String())
	}
	return nil
}
