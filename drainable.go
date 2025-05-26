package saving

import (
	"sync"
	"sync/atomic"
	"time"
)

type Status int

const (
	Drained Status = iota + 1
	waking
	Waked
	Failed
	draining
	rebooting
	terminated
)

func (s Status) GoString() string {
	switch s {
	case Drained:
		return "Drained"
	case waking:
		return "waking"
	case Waked:
		return "Waked"
	case Failed:
		return "Failed"
	case draining:
		return "draining"
	case rebooting:
		return "rebooting"
	case terminated:
		return "terminated"
	default:
		return "unknown"
	}
}

type Drainable struct {
	bootService  func() error
	closeService func() error
	drainTimeout time.Duration
	status       Status
	wait         chan struct{}
	lock         *sync.Mutex
	refCount     uint64
	error        error
	callback     func(s Status)
}

func NewDrainable(bootService, closeService func() error, drainTimeout time.Duration, callback func(s Status)) *Drainable {
	return &Drainable{
		bootService:  bootService,
		closeService: closeService,
		drainTimeout: drainTimeout,
		status:       Drained,
		wait:         make(chan struct{}),
		lock:         &sync.Mutex{},
		callback:     callback,
	}
}

func (d *Drainable) Exec(job func()) error {
	d.lock.Lock()
	switch d.status {
	case Drained:
		d.status = waking
		d.lock.Unlock()
		err := d.bootService()
		d.lock.Lock()
		if err == nil {
			d.status = Waked
			d.refCount++
		} else {
			d.status = Failed
			d.error = err
		}
		close(d.wait)
		d.wait = make(chan struct{})
		d.lock.Unlock()
		d.callback(d.status)
		if err == nil {
			job()
			atomic.AddUint64(&d.refCount, 1)
			time.AfterFunc(d.drainTimeout, d.timeout)
		}
	case Failed:
		d.lock.Unlock()
		return d.error
	case waking:
		d.lock.Unlock()
		<-d.wait
		if d.status == Waked {
			job()
			atomic.AddUint64(&d.refCount, 1)
			time.AfterFunc(d.drainTimeout, d.timeout)
		}
	case Waked:
		d.lock.Unlock()
		job()
		atomic.AddUint64(&d.refCount, 2)
		time.AfterFunc(d.drainTimeout, d.timeout)
	case draining:
		d.status = rebooting
		d.lock.Unlock()
		<-d.wait
		atomic.AddUint64(&d.refCount, 1)
	case rebooting:
		d.lock.Unlock()
	}
	return d.error
}

func (d *Drainable) Terminate() {
	// todo: implement terminate
	d.lock.Lock()
	defer d.lock.Unlock()
	d.status = terminated
}

func (d Drainable) IsWaking() bool {
	return d.status == Waked
}

func (d *Drainable) timeout() {
	d.lock.Lock()
	d.refCount -= 2
	if d.refCount > 0 {
		d.lock.Unlock()
		return
	}
	switch d.status {
	case Drained:
		d.lock.Unlock()
		panic("drainable: already drained")
	case waking:
		d.lock.Unlock()
		panic("drainable: counter is invalid")
	case terminated:
		d.lock.Unlock()
	case Waked:
		d.status = draining
		d.lock.Unlock()
		err := d.closeService()
		d.lock.Lock()
		switch d.status {
		case draining:
			if err == nil {
				d.status = Drained
			} else {
				d.status = Failed
				d.error = err
			}
			d.lock.Unlock()
			d.callback(d.status)
		case rebooting:
			if err == nil {
				d.lock.Unlock()
				d.bootService()
				d.lock.Lock()
				d.status = Waked
				d.refCount++
				close(d.wait)
				d.wait = make(chan struct{})
				d.lock.Unlock()
			} else {
				d.status = Failed
				d.error = err
				close(d.wait)
				d.wait = make(chan struct{})
				d.lock.Unlock()
			}

		default:
			panic("wrong status")
		}
	}
}
