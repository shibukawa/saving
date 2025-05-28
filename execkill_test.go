package saving

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestMain(m *testing.M) {
	cmd := exec.Command("go", "build")
	cmd.Dir = "./testdata/testserver"
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to build test server: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	os.Exit(code)
}

func getExecPath(t *testing.T) string {
	t.Helper()
	var ext string
	if "windows" == runtime.GOOS {
		ext = ".exe"
	}
	dir, _ := os.Getwd()
	execPath := filepath.Join(dir, "./testdata/testserver", "testserver"+ext)
	return execPath
}

func TestExecAndTerminate(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080/health")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, err := NewExecKillProcessController(ctx, ProcessOption{
		PidPath:            NormalizePidPath(""),
		HealthCheckUrl:     u,
		WakeTimeout:        time.Second,
		DrainTimeout:       time.Second,
		HealthCheckTimeout: time.Second,
		Cmd:                getExecPath(t),
		Args:               []string{},
	})
	called := false
	assert.NoError(t, err)
	err = p.Exec(func() {
		called = true
		res, err := http.Get("http://localhost:8080/hello")
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
	})
	assert.NoError(t, err)
	assert.True(t, called)
	assert.True(t, p.IsWaking())
	time.Sleep(2 * time.Second) // process is terminated
	assert.False(t, p.IsWaking())
}

func TestExecAgain(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080/health")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, err := NewExecKillProcessController(ctx, ProcessOption{
		PidPath:            NormalizePidPath(""),
		HealthCheckUrl:     u,
		WakeTimeout:        time.Second,
		DrainTimeout:       time.Second,
		HealthCheckTimeout: time.Second,
		Cmd:                getExecPath(t),
		Args:               []string{},
	})
	called := false
	assert.NoError(t, err)
	err = p.Exec(func() {
		called = true
		res, err := http.Get("http://localhost:8080/hello")
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
	})
	assert.NoError(t, err)
	assert.True(t, called)
	assert.True(t, p.IsWaking())
	initialPid := p.Pid()
	time.Sleep(2 * time.Second) // process is terminated
	assert.False(t, p.IsWaking())

	called2 := false
	// call again: sleepable process kick new process for the next exec request
	err = p.Exec(func() {
		called2 = true
		res, err := http.Get("http://localhost:8080/hello")
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
	})
	assert.NoError(t, err)
	assert.True(t, called2)
	assert.True(t, p.IsWaking())

	assert.NotEqual(t, initialPid, p.Pid())
}
