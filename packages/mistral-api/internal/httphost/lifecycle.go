package httphost

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

func Run(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	if ctx == nil {
		return errors.New("host context is required")
	}
	if server == nil {
		return errors.New("http server is required")
	}
	if listener == nil {
		return errors.New("http listener is required")
	}
	if shutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown http server: %w", err)
		}
		if err := <-serveErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
