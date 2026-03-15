package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/task"
)

func TestManagerTracksLifecycle(t *testing.T) {
	manager := task.NewManager(1)
	defer func() {
		_ = manager.Close()
	}()

	started := make(chan struct{})
	finish := make(chan struct{})

	id := manager.Submit("demo", func(context.Context) error {
		close(started)
		<-finish
		return nil
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}

	info, ok := manager.Status(id)
	if !ok {
		t.Fatal("task status should exist")
	}
	if info.Status != task.StatusRunning {
		t.Fatalf("expected running, got %s", info.Status)
	}

	close(finish)
	waitForStatus(t, manager, id, task.StatusSucceeded)
}

func TestManagerMarksFailure(t *testing.T) {
	manager := task.NewManager(1)
	defer func() {
		_ = manager.Close()
	}()

	id := manager.Submit("failing", func(context.Context) error {
		return errors.New("boom")
	})

	waitForStatus(t, manager, id, task.StatusFailed)
}

func waitForStatus(t *testing.T, manager *task.Manager, id string, expected task.Status) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		info, ok := manager.Status(id)
		if ok && info.Status == expected {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	info, _ := manager.Status(id)
	t.Fatalf("expected status %s, got %+v", expected, info)
}
