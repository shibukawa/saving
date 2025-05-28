package saving

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"time"
)

type CriuProcessController struct {
	drainable *Drainable
	access    uint64
	pid       int
	ProcessOption
}

// IsWaking implements ProcessController.
func (c *CriuProcessController) IsWaking() bool {
	panic("unimplemented")
}

// Pid implements ProcessController.
func (c *CriuProcessController) Pid() int {
	return c.pid
}

var _ ProcessController = (*CriuProcessController)(nil)

func NewCriuProcessController(ctx context.Context, opt ProcessOption) (*CriuProcessController, error) {
	err := writePid(opt.PidPath, nil)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(opt.CriuDumpPath, 0o666)

	result := &CriuProcessController{
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

	// exec process
	cmd := exec.Command(opt.Cmd, opt.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	result.pid = cmd.Process.Pid
	opt.Logger.Info("process start", "pid", result.pid)
	status := WaitAndCheckHealth(result.WakeTimeout, result.HealthCheckUrl)
	if !status {
		return nil, ErrHealthCheckFailed
	}
	err = result.stop()
	if err != nil {
		return nil, err
	}

	// force stop process when context is done
	go func() {
		<-ctx.Done()
		result.stop()
		os.Remove(opt.PidPath)
	}()

	return result, nil
}

// Exec implements ProcessController.
func (c *CriuProcessController) Exec(callback func()) error {
	atomic.AddUint64(&c.access, 1)
	return c.drainable.Exec(callback)
}

func (c *CriuProcessController) start() error {
	atomic.StoreUint64(&c.access, 0)
	start := time.Now()
	cmd := exec.Command(c.CriuPath, "restore", "-D", c.CriuDumpPath)
	result, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	c.Logger.Info("process start by criu", slog.Duration("boot_time", time.Now().Sub(start)))
	c.Logger.Info(string(result))
	/*status := WaitAndCheckHealth(c.WakeTimeout, c.HealthCheckUrl)
	if !status {
		return ErrHealthCheckFailed
	}*/
	return writePid(c.PidPath, c.HealthCheckUrl)
}

func (c *CriuProcessController) stop() error {
	c.Logger.Info("process stop", "pid", c.pid, "access", c.access)
	writePid(c.PidPath, nil)
	cmd := exec.Command(c.CriuPath, "dump", "--shell-job", "-t", strconv.Itoa(c.pid), "-D", c.CriuDumpPath)
	result, err := cmd.CombinedOutput()
	c.Logger.Info(string(result))
	if err != nil {
		return err
	}
	return nil
}
