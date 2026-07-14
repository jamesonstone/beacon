package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"
)

var ErrUnavailable = errors.New("Beacon agent unavailable")

type Client struct {
	Socket  string
	Timeout time.Duration
}

func (c Client) Request(ctx context.Context, request Request) (Event, error) {
	connection, err := c.dial(ctx)
	if err != nil {
		return Event{}, err
	}
	defer connection.Close()
	if c.Timeout > 0 {
		if err := connection.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
			return Event{}, fmt.Errorf("set agent request deadline: %w", err)
		}
	} else if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return Event{}, fmt.Errorf("set agent request deadline: %w", err)
		}
	}
	request.ProtocolVersion = ProtocolVersion
	if request.RequestID == "" {
		request.RequestID = newID()
	}
	if err := Encode(connection, request); err != nil {
		return Event{}, fmt.Errorf("send agent request: %w", err)
	}
	event, err := NewEventDecoder(connection).Next()
	if err != nil {
		return Event{}, err
	}
	if event.RequestID != request.RequestID {
		return Event{}, fmt.Errorf("agent response request_id mismatch: %s", event.RequestID)
	}
	return event, nil
}

func (c Client) Subscribe(ctx context.Context) (<-chan Event, <-chan error, error) {
	connection, err := c.dial(ctx)
	if err != nil {
		return nil, nil, err
	}
	request := Request{ProtocolVersion: ProtocolVersion, RequestID: newID(), Type: RequestSubscribe}
	if err := Encode(connection, request); err != nil {
		connection.Close()
		return nil, nil, fmt.Errorf("subscribe to agent: %w", err)
	}
	events := make(chan Event, 16)
	errors := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errors)
		defer connection.Close()
		decoder := NewEventDecoder(connection)
		for {
			event, err := decoder.Next()
			if err != nil {
				errors <- err
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, errors, nil
}

func (c Client) dial(ctx context.Context) (net.Conn, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	connection, err := dialer.DialContext(ctx, "unix", c.Socket)
	if err != nil {
		return nil, fmt.Errorf("%w: connect at %s: %v", ErrUnavailable, c.Socket, err)
	}
	return connection, nil
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}
