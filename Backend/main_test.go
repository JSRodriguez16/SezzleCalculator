package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInvalidPortFailsBeforeListening(t *testing.T) {
	for _, port := range []string{"0", "65536", "-1", "eight"} {
		t.Run(port, func(t *testing.T) {
			t.Setenv("PORT", port)
			if err := run(); err == nil || !strings.Contains(err.Error(), "PORT") {
				t.Fatalf("run() = %v, want PORT validation error", err)
			}
		})
	}
}

func TestInvalidStaticDirectoryFailsBeforeListening(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("STATIC_DIR", filepath.Join(t.TempDir(), "missing"))
	if err := run(); err == nil || !strings.Contains(err.Error(), "static frontend") {
		t.Fatalf("run() = %v, want static configuration error", err)
	}
}

func TestMainReportsRunFailure(t *testing.T) {
	t.Setenv("PORT", "invalid")
	previousExit := processExit
	defer func() { processExit = previousExit }()
	exitCode := 0
	processExit = func(code int) { exitCode = code }

	main()
	if exitCode != 1 {
		t.Fatalf("processExit code = %d, want 1", exitCode)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	t.Setenv("PORT", strconv.Itoa(port))
	if err := run(); err == nil {
		t.Fatal("run() should return when the port is already in use")
	}
}

func TestMainShutsDownOnInterrupt(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	t.Setenv("PORT", strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		runWithContext(ctx)
		close(done)
	}()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/api/health"
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, requestErr := client.Get(url)
		if requestErr == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("main() did not shut down after interrupt")
	}
}

func TestRunWithContextReturnsShutdownError(t *testing.T) {
	previousShutdown := shutdownServer
	defer func() { shutdownServer = previousShutdown }()
	shutdownServer = func(*http.Server, context.Context) error { return errors.New("shutdown failure") }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runWithContext(ctx); err == nil || !strings.Contains(err.Error(), "graceful shutdown") {
		t.Fatalf("runWithContext() error = %v, want graceful shutdown error", err)
	}
}

func TestRunWithContextReturnsUnexpectedServerError(t *testing.T) {
	previousListen, previousShutdown := listenAndServe, shutdownServer
	defer func() { listenAndServe, shutdownServer = previousListen, previousShutdown }()
	serverError := errors.New("unexpected server error")
	listenAndServe = func(*http.Server) error {
		time.Sleep(10 * time.Millisecond)
		return serverError
	}
	shutdownServer = func(*http.Server, context.Context) error { return nil }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runWithContext(ctx); !errors.Is(err, serverError) {
		t.Fatalf("runWithContext() error = %v, want %v", err, serverError)
	}
}
