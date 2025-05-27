package saving

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// StartProxy is a main function of this package.
func StartProxy(ctx context.Context, opt Option) error {
	process, err := NewProcess(ctx, opt.ToProcessOption())
	if err != nil {
		return err
	}

	for _, p := range opt.PortMaps {
		NewSingleProxyServer(ctx, process, p.FromPort, p.Destination)
	}
	<-ctx.Done()
	os.Remove(opt.PidPath)
	return nil
}

func NewSingleProxyServer(ctx context.Context, process *Process, listeningPort string, dest *url.URL) *http.Server {
	server := &http.Server{
		Addr: listeningPort,
		Handler: &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				process.Exec(func() {
					r.SetURL(dest)
					r.SetXForwarded()
				})
			},
		},
	}
	go func() {
		server.ListenAndServe()
	}()
	go func() {
		<-ctx.Done()
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx2); err != nil {
			panic(err)
		}
	}()
	return server
}

func CheckProcessHealth(PidPath string) bool {
	content, err := os.ReadFile(PidPath)
	if os.IsNotExist(err) {
		return false
	}
	chunks := bytes.SplitN(content, []byte{':'}, 2)
	if len(chunks) == 1 { // only saving process is working
		return true
	}
	u, _ := url.Parse(string(chunks[1]))
	return CheckHealth(u)
}
