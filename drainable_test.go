package saving

import (
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func wait(wait time.Duration) func() error {
	return func() error {
		time.Sleep(wait)
		return nil
	}
}

var ErrBoot = errors.New("error boot")
var ErrClose = errors.New("error close")

func failAfter(wait time.Duration, err error) func() error {
	return func() error {
		time.Sleep(wait)
		return err
	}
}

func TestWake(t *testing.T) {
	var currentStatus Status
	drainable := NewDrainable(wait(100*time.Millisecond), wait(0), 100*time.Millisecond, func(s Status) {
		currentStatus = s
	})
	assert.False(t, drainable.IsWaking())
	wg := &sync.WaitGroup{}
	wg.Add(5)
	for range 5 {
		go func() {
			defer wg.Done()
			err := drainable.Exec(func() {})
			assert.True(t, drainable.IsWaking())
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.Equal(t, Waked, currentStatus)
}

func TestDrained(t *testing.T) {
	var currentStatus Status
	timeout := 100 * time.Millisecond
	drainable := NewDrainable(wait(100*time.Millisecond), wait(0), timeout, func(s Status) {
		currentStatus = s
	})
	assert.False(t, drainable.IsWaking())
	err := drainable.Exec(func() {})
	assert.True(t, drainable.IsWaking())
	assert.NoError(t, err)
	assert.Equal(t, Waked, currentStatus)

	time.Sleep(2 * timeout) // job is drained after timeout duration
	assert.False(t, drainable.IsWaking())
	assert.Equal(t, Drained, currentStatus)
}

func TestStartJobDuringDraining(t *testing.T) {
	var currentStatus Status
	timeout := 100 * time.Millisecond
	drainable := NewDrainable(wait(100*time.Millisecond), wait(100*time.Millisecond), timeout, func(s Status) {
		currentStatus = s
	})
	assert.False(t, drainable.IsWaking())
	drainable.Exec(func() {})
	assert.True(t, drainable.IsWaking())
	assert.Equal(t, Waked, currentStatus)

	time.Sleep(150 * time.Millisecond) // job is start draining
	assert.False(t, drainable.IsWaking())
	drainable.Exec(func() {})
	assert.True(t, drainable.IsWaking())
	assert.Equal(t, Waked, currentStatus)
}

func TestFailedToStart(t *testing.T) {
	var currentStatus Status
	timeout := 100 * time.Millisecond
	drainable := NewDrainable(failAfter(100*time.Millisecond, ErrBoot), wait(0), timeout, func(s Status) {
		currentStatus = s
	})
	assert.False(t, drainable.IsWaking())
	err := drainable.Exec(func() {})
	assert.IsError(t, err, ErrBoot)
	assert.False(t, drainable.IsWaking())
	assert.Equal(t, Failed, currentStatus)
}

func TestFailedToClose(t *testing.T) {
	var currentStatus Status
	timeout := 100 * time.Millisecond
	drainable := NewDrainable(wait(100*time.Millisecond), failAfter(100*time.Millisecond, ErrClose), timeout, func(s Status) {
		currentStatus = s
	})
	assert.False(t, drainable.IsWaking())
	err := drainable.Exec(func() {})
	assert.NoError(t, err)
	assert.True(t, drainable.IsWaking())
	assert.Equal(t, Waked, currentStatus)

	time.Sleep(3 * timeout) // job is start draining
	assert.False(t, drainable.IsWaking())
	assert.Equal(t, Failed, currentStatus)
	err2 := drainable.Exec(func() {})
	assert.IsError(t, ErrClose, err2)
}
