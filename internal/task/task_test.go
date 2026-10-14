package task

import (
	"testing"
	"time"
)

func TestCertificateTaskCompletesWithoutLogConsumer(t *testing.T) {
	task := New("cert-example")
	finished := make(chan struct{})
	go func() {
		for i := 0; i < 1100; i++ {
			task.Log("INFO", "progress")
		}
		task.Done()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("certificate task blocked without a browser log consumer")
	}
	status := task.Snapshot(0)
	if !status.Done || status.Next != 1100 || len(status.Logs) != 1000 {
		t.Fatalf("unexpected status: %+v", status)
	}
	if next := task.Snapshot(status.Next); len(next.Logs) != 0 || !next.Done {
		t.Fatal("completed task cannot be read again")
	}
}

func TestTaskResultCanBeReadAfterDisconnect(t *testing.T) {
	task := New("cert-example")
	task.Log("ERROR", "验证失败")
	task.Done()
	task.Done()
	for i := 0; i < 2; i++ {
		status := task.Snapshot(0)
		if !status.Done || status.Error != "验证失败" || len(status.Logs) != 1 {
			t.Fatalf("lost task result: %+v", status)
		}
	}
	if duplicate := New("cert-example"); duplicate.ID == task.ID {
		t.Fatal("new task reused an old task ID")
	}
}
