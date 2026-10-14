package task

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type Task struct {
	ID   string
	ch   chan string
	done bool
	logs []LogLine
	next int
	err  string
	mu   sync.Mutex
}

type LogLine struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type Status struct {
	Logs  []LogLine `json:"logs"`
	Next  int       `json:"next"`
	Done  bool      `json:"done"`
	Error string    `json:"error"`
}

var (
	tasks = map[string]*Task{}
	mu    sync.Mutex
)

func New(id string) *Task {
	t := &Task{
		ID: id + "-" + rand.Text(),
		ch: make(chan string, 100),
	}
	mu.Lock()
	tasks[t.ID] = t
	mu.Unlock()
	return t
}

func Get(id string) *Task {
	mu.Lock()
	defer mu.Unlock()
	return tasks[id]
}

func (t *Task) Log(level, msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return
	}
	entry := LogLine{Time: time.Now().Format("2006-01-02 15:04:05"), Level: level, Message: msg}
	t.logs = append(t.logs, entry)
	t.next++
	if len(t.logs) > 1000 {
		t.logs = append([]LogLine(nil), t.logs[len(t.logs)-1000:]...)
	}
	if level == "ERROR" {
		t.err = msg
	}
	// Log delivery must never prevent certificate issuance when the browser disconnects.
	select {
	case t.ch <- entry.Time + " " + level + " " + msg:
	default:
	}
}

func (t *Task) Snapshot(after int) Status {
	t.mu.Lock()
	defer t.mu.Unlock()
	start := after - (t.next - len(t.logs))
	if start < 0 || start > len(t.logs) {
		start = 0
	}
	logs := append([]LogLine{}, t.logs[start:]...)
	return Status{Logs: logs, Next: t.next, Done: t.done, Error: t.err}
}

func (t *Task) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return
	}
	t.done = true
	close(t.ch)
	time.AfterFunc(10*time.Minute, func() {
		mu.Lock()
		delete(tasks, t.ID)
		mu.Unlock()
	})
}

func (t *Task) Chan() <-chan string {
	return t.ch
}

// ── Logger ────────────────────────────────────────────────────────────────────

type Logger struct {
	t *Task
}

func NewLogger(t *Task) *Logger {
	return &Logger{t: t}
}

func (l *Logger) Fatal(args ...interface{}) {
	l.t.Log("ERROR", fmt.Sprint(args...))
}

func (l *Logger) Fatalln(args ...interface{}) {
	l.t.Log("ERROR", fmt.Sprintln(args...))
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.t.Log("ERROR", fmt.Sprintf(format, args...))
}

func (l *Logger) Print(args ...interface{}) {
	l.t.Log("INFO", fmt.Sprint(args...))
}

func (l *Logger) Println(args ...interface{}) {
	l.t.Log("INFO", fmt.Sprintln(args...))
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.t.Log("INFO", fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.t.Log("INFO", fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.t.Log("WARN", fmt.Sprintf(format, args...))
}
