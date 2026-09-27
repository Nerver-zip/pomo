package db

import (
	"errors"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
)

var (
	ErrTaskNotFound   = errors.New("task not found")
	ErrEmptyTaskTitle = errors.New("task title cannot be empty")
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskCompleted TaskStatus = "completed"
)

// Task represents a persistent unit of focused work.
type Task struct {
	ID          int        `db:"id" json:"id"`
	Title       string     `db:"title" json:"title"`
	Description string     `db:"description" json:"description"`
	Status      TaskStatus `db:"status" json:"status"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Computed aggregate metrics
	TotalPomodoros int           `db:"total_pomodoros" json:"total_pomodoros"`
	TotalDuration  time.Duration `db:"total_duration" json:"total_duration"`
}

// Session represents a recorded block of focus or rest time.
type Session struct {
	ID        int           `db:"id" json:"id"`
	TaskID    *int          `db:"task_id" json:"task_id,omitempty"`
	Type      string        `db:"type" json:"type"`
	Duration  time.Duration `db:"duration" json:"duration"`
	StartedAt string        `db:"started_at" json:"started_at"`
}

func (s Session) StartedAtTime() time.Time {
	t, _ := parseTime(s.StartedAt)
	return t
}

type TaskStats struct {
	TaskID         int           `db:"task_id" json:"task_id"`
	Title          string        `db:"title" json:"title"`
	Status         TaskStatus    `db:"status" json:"status"`
	CreatedAt      time.Time     `db:"created_at" json:"created_at"`
	CompletedAt    *time.Time    `db:"completed_at" json:"completed_at,omitempty"`
	TotalPomodoros int           `db:"total_pomodoros" json:"total_pomodoros"`
	TotalDuration  time.Duration `db:"total_duration" json:"total_duration"`
}

type AllTimeStats struct {
	TotalSessions      int           `db:"total_sessions"`
	TotalWorkDuration  time.Duration `db:"total_work_duration"`
	TotalBreakDuration time.Duration `db:"total_break_duration"`
}

type DailyStat struct {
	Date         string        `db:"day"`
	WorkDuration time.Duration `db:"work_duration"`
}

type StreakStats struct {
	Current int
	Best    int
}

type SessionType string

const (
	WorkSession  SessionType = "work"
	BreakSession SessionType = "break"
)

func GetSessionType(taskType config.TaskType) SessionType {
	if taskType == config.WorkTask {
		return WorkSession
	}
	return BreakSession
}
