package saving

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func httpStatusCodeServer(t *testing.T, statusCodes []int) (port uint16, close func()) {
	t.Helper()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to listen on a random port: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	port = uint16(addr.Port)

	start := time.Now()
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			after := int(time.Now().Sub(start) / time.Second)
			if after >= len(statusCodes) {
				after = len(statusCodes) - 1
			}
			w.WriteHeader(statusCodes[after])
		}),
	}
	server.RegisterOnShutdown(func() {
		listener.Close()
	})
	go server.Serve(listener)

	return port, func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx2)
	}
}

func TestQuickCheckHealthy(t *testing.T) {
	port, close := httpStatusCodeServer(t, []int{200})
	defer close()
	u, _ := url.Parse(fmt.Sprintf("http://localhost:%d/health", port))
	status := WaitAndCheckHealth(time.Second, u)
	assert.Equal(t, true, status)
}

func TestQuickCheckUnealthy(t *testing.T) {
	port, close := httpStatusCodeServer(t, []int{500})
	defer close()
	u, _ := url.Parse(fmt.Sprintf("http://localhost:%d/health", port))
	status := WaitAndCheckHealth(time.Second, u)
	assert.Equal(t, false, status)
}
