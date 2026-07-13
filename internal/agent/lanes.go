package agent

import (
	"errors"

	"github.com/jamesonstone/beacon/internal/model"
)

func (e *Engine) SetLaneAttention(id string, state model.AttentionState) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.SetAttention(e.Snapshot(), id, state)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", string(state), &snapshot)
	return nil
}

func (e *Engine) SetLanePinned(id string, pinned bool) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.SetPinned(e.Snapshot(), id, pinned)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "pin updated", &snapshot)
	return nil
}

func (e *Engine) SetLaneNote(id, note string) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.SetNote(e.Snapshot(), id, note)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "note updated", &snapshot)
	return nil
}

func (e *Engine) AddLaneTag(id, tag string) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.AddTag(e.Snapshot(), id, tag)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "tag added", &snapshot)
	return nil
}

func (e *Engine) RemoveLaneTag(id, tag string) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.RemoveTag(e.Snapshot(), id, tag)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "tag removed", &snapshot)
	return nil
}

func (e *Engine) MarkLaneSeen(id string) error {
	if e.WorkingSet == nil {
		return errors.New("working-set authority is unavailable")
	}
	snapshot, err := e.WorkingSet.MarkSeen(e.Snapshot(), id)
	if err != nil {
		return err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "marked seen", &snapshot)
	return nil
}

func (e *Engine) AddManualLane(title string) (string, error) {
	if e.WorkingSet == nil {
		return "", errors.New("working-set authority is unavailable")
	}
	snapshot, id, err := e.WorkingSet.AddManual(e.Snapshot(), title)
	if err != nil {
		return "", err
	}
	e.publish(EventWorkingSetChanged, "", id, 0, "ready", "manual lane added", &snapshot)
	return id, nil
}
