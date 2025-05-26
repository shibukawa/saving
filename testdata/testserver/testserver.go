package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(">>> hello")
		defer fmt.Println("<<< hello")
		fmt.Fprintf(w, "hello world")
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	srv := &http.Server{
		Addr: ":8080",
	}
	go func() {
		log.Printf("start listening at %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx2); err != nil {
		log.Fatalf("Fail to shutdown: %v", err)
	}
}
