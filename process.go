package saving

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

var ErrHealthCheckFailed = errors.New("health check failed")

const DefaultPidFilename = "SAVING_PID"

type Process struct {
	drainable *Drainable
	pid       int
	access    uint64
	ProcessOption
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

func NewProcess(ctx context.Context, opt ProcessOption) (*Process, error) {
	err := writePid(opt.PidPath, nil)
	if err != nil {
		return nil, err
	}

	result := &Process{
		ProcessOption: opt,
	}

	drainable := NewDrainable(
		result.start,
		result.stop,
		opt.DrainTimeout,
		func(s Status) {
			switch s {
			case Failed:
				os.Remove(opt.PidPath)
			}
		},
	)

	result.drainable = drainable

	// force stop process when context is done
	go func() {
		<-ctx.Done()
		result.stop()
		os.Remove(opt.PidPath)
	}()

	return result, nil
}

func (p *Process) Exec(callback func()) error {
	atomic.AddUint64(&p.
		access, 1)
	return p.drainable.Exec(callback)
}

func (p *Process) IsWaking() bool {
	return p.drainable.IsWaking()
}

func (p Process) Pid() int {
	return p.pid
}

func (p *Process) start() error {
	atomic.StoreUint64(&p.access, 0)
	cmd := exec.Command(p.Cmd, p.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	p.pid = cmd.Process.Pid

	p.Logger.Info("process start", "pid", p.pid)

	status := WaitAndCheckHealth(p.WakeTimeout, p.HealthCheckUrl)
	if !status {
		return ErrHealthCheckFailed
	}
	return writePid(p.PidPath, p.HealthCheckUrl)
}

func (p *Process) stop() error {
	p.Logger.Info("process stop", "pid", p.pid, "access", p.access)
	os.Remove(p.PidPath)
	process, err := os.FindProcess(p.pid)
	if err != nil {
		return err // already terminated
	}
	// send sigterm
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	done := make(chan struct{})

	go func() {
		process.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// send sigkill
		process.Signal(syscall.SIGKILL)
	}
	return nil
}

func NormalizePidPath(pidPath string) string {
	if pidPath == "" {
		return filepath.Join(os.TempDir(), DefaultPidFilename)
	}
	return pidPath
}
