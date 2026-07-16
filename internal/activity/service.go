package activity

import (
	"context"
	"io"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
)

type AgentClient interface {
	Request(context.Context, agent.Request) (agent.Event, error)
}

type Service struct {
	Store    Store
	Agent    AgentClient
	Observe  func(string) error
	Now      func() time.Time
	Deadline time.Duration
}

func (s Service) Ingest(ctx context.Context, provider string, input io.Reader) error {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	event, err := Decode(provider, input, now)
	if err != nil {
		return err
	}
	if s.Observe != nil {
		if err := s.Observe(provider); err != nil {
			return err
		}
	}
	if event.Action == ActionNone || s.Agent == nil {
		return nil
	}
	deadline := s.Deadline
	if deadline <= 0 || deadline >= 500*time.Millisecond {
		deadline = 450 * time.Millisecond
	}
	workContext, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	snapshotEvent, err := s.Agent.Request(workContext, agent.Request{Type: agent.RequestGetSnapshot})
	if err != nil || snapshotEvent.Snapshot == nil {
		return nil
	}
	target, err := Match(*snapshotEvent.Snapshot, event.CWD)
	if err != nil {
		return nil
	}
	_, refresh, err := s.Store.Apply(workContext, event, target, now)
	if err != nil {
		return err
	}
	if refresh {
		_, _ = s.Agent.Request(workContext, agent.Request{Type: agent.RequestRefreshProject, ProjectID: target.ProjectID})
	}
	return nil
}
