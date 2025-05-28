package saving

import (
	"context"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

type ExecKillProcessController struct {
	drainable *Drainable
	pid       int
	access    uint64
	ProcessOption
}

var _ ProcessController = (*ExecKillProcessController)(nil)

func NewExecKillProcessController(ctx context.Context, opt ProcessOption) (*ExecKillProcessController, error) {
	err := writePid(opt.PidPath, nil)
	if err != nil {
		return nil, err
	}

	result := &ExecKillProcessController{
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

func (p *ExecKillProcessController) Exec(callback func()) error {
	atomic.AddUint64(&p.access, 1)
	return p.drainable.Exec(callback)
}

func (p *ExecKillProcessController) IsWaking() bool {
	return p.drainable.IsWaking()
}

func (p ExecKillProcessController) Pid() int {
	return p.pid
}

func (p *ExecKillProcessController) start() error {
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

func (p *ExecKillProcessController) stop() error {
	p.Logger.Info("process stop", "pid", p.pid, "access", p.access)
	writePid(p.PidPath, nil)
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
