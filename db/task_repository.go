package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type TaskRepo struct {
	db *sqlx.DB
}

func NewTaskRepo(db *sqlx.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// Create inserts a new pending task into the database.
func (r *TaskRepo) Create(title, description string) (*Task, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return nil, ErrEmptyTaskTitle
	}
	cleanDesc := strings.TrimSpace(description)

	createdAt := time.Now().UTC()
	createdAtStr := createdAt.Format(time.RFC3339)

	res, err := r.db.Exec(`
		INSERT INTO tasks (title, description, status, created_at)
		VALUES (?, ?, ?, ?);
	`, cleanTitle, cleanDesc, string(TaskPending), createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve inserted task ID: %w", err)
	}

	return &Task{
		ID:          int(id),
		Title:       cleanTitle,
		Description: cleanDesc,
		Status:      TaskPending,
		CreatedAt:   createdAt,
	}, nil
}

// GetByID retrieves a task by its primary key ID.
func (r *TaskRepo) GetByID(id int) (*Task, error) {
	var row struct {
		ID             int            `db:"id"`
		Title          string         `db:"title"`
		Description    string         `db:"description"`
		Status         string         `db:"status"`
		CreatedAt      string         `db:"created_at"`
		CompletedAt    sql.NullString `db:"completed_at"`
		TotalPomodoros int            `db:"total_pomodoros"`
		TotalDuration  int64          `db:"total_duration"`
	}

	err := r.db.Get(&row, `
		SELECT t.id, t.title, t.description, t.status, t.created_at, t.completed_at,
		       COUNT(s.id) AS total_pomodoros,
		       COALESCE(SUM(s.duration), 0) AS total_duration
		FROM tasks t
		LEFT JOIN sessions s ON s.task_id = t.id AND s.type = 'work'
		WHERE t.id = ?
		GROUP BY t.id;
	`, id)
	if err != nil {
		if errorsIs(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to query task %d: %w", id, err)
	}

	t, err := parseTaskRow(row.ID, row.Title, row.Description, row.Status, row.CreatedAt, row.CompletedAt, row.TotalPomodoros, row.TotalDuration)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListPending retrieves all pending tasks ordered by ID ascending.
func (r *TaskRepo) ListPending() ([]Task, error) {
	return r.listTasks(false)
}

// ListAll retrieves all tasks, optionally including completed tasks.
func (r *TaskRepo) ListAll(includeCompleted bool) ([]Task, error) {
	return r.listTasks(includeCompleted)
}

func (r *TaskRepo) listTasks(includeCompleted bool) ([]Task, error) {
	query := `
		SELECT t.id, t.title, t.description, t.status, t.created_at, t.completed_at,
		       COUNT(s.id) AS total_pomodoros,
		       COALESCE(SUM(s.duration), 0) AS total_duration
		FROM tasks t
		LEFT JOIN sessions s ON s.task_id = t.id AND s.type = 'work'
	`
	if !includeCompleted {
		query += " WHERE t.status = 'pending'"
	}
	query += " GROUP BY t.id ORDER BY t.id ASC;"

	type rowStruct struct {
		ID             int            `db:"id"`
		Title          string         `db:"title"`
		Description    string         `db:"description"`
		Status         string         `db:"status"`
		CreatedAt      string         `db:"created_at"`
		CompletedAt    sql.NullString `db:"completed_at"`
		TotalPomodoros int            `db:"total_pomodoros"`
		TotalDuration  int64          `db:"total_duration"`
	}

	var rows []rowStruct
	if err := r.db.Select(&rows, query); err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		t, err := parseTaskRow(row.ID, row.Title, row.Description, row.Status, row.CreatedAt, row.CompletedAt, row.TotalPomodoros, row.TotalDuration)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}

	return tasks, nil
}

// Update modifies title and description of a task.
func (r *TaskRepo) Update(id int, title, description string) error {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return ErrEmptyTaskTitle
	}
	cleanDesc := strings.TrimSpace(description)

	res, err := r.db.Exec(`
		UPDATE tasks
		SET title = ?, description = ?
		WHERE id = ?;
	`, cleanTitle, cleanDesc, id)
	if err != nil {
		return fmt.Errorf("failed to update task %d: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Complete marks a task as completed with current timestamp.
func (r *TaskRepo) Complete(id int) error {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(`
		UPDATE tasks
		SET status = ?, completed_at = ?
		WHERE id = ?;
	`, string(TaskCompleted), nowStr, id)
	if err != nil {
		return fmt.Errorf("failed to complete task %d: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Reopen marks a completed task back to pending and clears completed_at.
func (r *TaskRepo) Reopen(id int) error {
	res, err := r.db.Exec(`
		UPDATE tasks
		SET status = ?, completed_at = NULL
		WHERE id = ?;
	`, string(TaskPending), id)
	if err != nil {
		return fmt.Errorf("failed to reopen task %d: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Delete removes a task. Foreign key ON DELETE SET NULL disassociates sessions.
func (r *TaskRepo) Delete(id int) error {
	res, err := r.db.Exec("DELETE FROM tasks WHERE id = ?;", id)
	if err != nil {
		return fmt.Errorf("failed to delete task %d: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// GetTaskStats aggregates focus metrics for a single task.
func (r *TaskRepo) GetTaskStats(id int) (*TaskStats, error) {
	task, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	return &TaskStats{
		TaskID:         task.ID,
		Title:          task.Title,
		Status:         task.Status,
		CreatedAt:      task.CreatedAt,
		CompletedAt:    task.CompletedAt,
		TotalPomodoros: task.TotalPomodoros,
		TotalDuration:  task.TotalDuration,
	}, nil
}

func parseTaskRow(id int, title, desc, status, createdAtStr string, completedAtStr sql.NullString, pomodoros int, durationNano int64) (*Task, error) {
	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		createdAt = time.Now()
	}

	var completedAt *time.Time
	if completedAtStr.Valid && completedAtStr.String != "" {
		if ct, err := parseTime(completedAtStr.String); err == nil {
			completedAt = &ct
		}
	}

	return &Task{
		ID:             id,
		Title:          title,
		Description:    desc,
		Status:         TaskStatus(status),
		CreatedAt:      createdAt,
		CompletedAt:    completedAt,
		TotalPomodoros: pomodoros,
		TotalDuration:  time.Duration(durationNano),
	}, nil
}

func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown time format: %q", s)
}

func errorsIs(err, target error) bool {
	return err == target
}
