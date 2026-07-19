package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func (s *Server) Serve(ctx context.Context) error {
	if s.Engine == nil {
		return errors.New("agent engine is required")
	}
	if err := s.Paths.EnsureRuntime(); err != nil {
		return err
	}
	release, err := acquirePIDLock(s.Paths.PID)
	if err != nil {
		return err
	}
	defer release()
	_ = os.Remove(s.Paths.Socket)
	listener, err := listenUnixSocket(ctx, s.Paths.Socket)
	if err != nil {
		return fmt.Errorf("listen on agent socket %s: %w", s.Paths.Socket, err)
	}
	defer listener.Close()
	defer os.Remove(s.Paths.Socket)
	if err := os.Chmod(s.Paths.Socket, 0o600); err != nil {
		return fmt.Errorf("secure agent socket: %w", err)
	}
	serverContext, cancel := context.WithCancel(ctx)
	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()
	defer cancel()
	go func() {
		<-serverContext.Done()
		_ = listener.Close()
	}()
	go s.Engine.RunSchedule(serverContext)
	go s.heartbeats(serverContext)

	for {
		connection, err := listener.Accept()
		if err != nil {
			if serverContext.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept agent connection: %w", err)
		}
		go s.handle(serverContext, connection)
	}
}

func listenUnixSocket(ctx context.Context, path string) (net.Listener, error) {
	previousMask := unix.Umask(0o077)
	defer unix.Umask(previousMask)
	return (&net.ListenConfig{}).Listen(ctx, "unix", path)
}

func (s *Server) Stop() {
	s.mutex.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mutex.Unlock()
}
