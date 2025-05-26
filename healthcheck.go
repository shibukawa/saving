package saving

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

var ErrOption = errors.New("option error")

func WaitAndCheckHealth(timeout time.Duration, target *url.URL) bool {
	start := time.Now()
	initialTicker := time.NewTicker(100 * time.Millisecond)
	for now := range initialTicker.C {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if res, err := http.DefaultClient.Do(req); err == nil && res.StatusCode == http.StatusOK {
			initialTicker.Stop()
			return true
		} else if now.Sub(start) > timeout {
			break
		}
	}
	return false
}

func CheckHealth(target *url.URL) bool {
	if res, err := http.Get(target.String()); err == nil && res.StatusCode == http.StatusOK {
		return true
	}
	return false
}
