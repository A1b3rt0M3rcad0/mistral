package httphost

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestRunServesAndShutsDownCleanly(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}),
		ReadHeaderTimeout: time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, server, listener, time.Second)
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String())
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not shut down")
	}
}

func TestRunValidatesDependencies(t *testing.T) {
	server := &http.Server{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	if err := Run(nil, server, listener, time.Second); err == nil {
		t.Fatal("expected missing context error")
	}
	if err := Run(context.Background(), nil, listener, time.Second); err == nil {
		t.Fatal("expected missing server error")
	}
	if err := Run(context.Background(), server, nil, time.Second); err == nil {
		t.Fatal("expected missing listener error")
	}
	if err := Run(context.Background(), server, listener, 0); err == nil {
		t.Fatal("expected invalid shutdown timeout error")
	}
}
