package ui

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/db"
	"github.com/Nerver-zip/pomo-tasker/ui/ascii"
	"github.com/Nerver-zip/pomo-tasker/ui/colors"
	"github.com/Nerver-zip/pomo-tasker/ui/confirm"
	"github.com/Nerver-zip/pomo-tasker/ui/summary"
	"github.com/Nerver-zip/pomo-tasker/ui/taskform"
	"github.com/Nerver-zip/pomo-tasker/ui/taskpicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type Model struct {
	// components
	progressBar   progress.Model
	confirmDialog confirm.Model
	help          help.Model
	picker        taskpicker.Model
	form          taskform.Model

	// timer
	timer    timer.Model
	duration time.Duration
	elapsed  time.Duration

	// state
	width, height    int // window dimensions
	onSessionEnd     string
	sessionState     SessionState
	prevState        SessionState
	confirmStartTime time.Time
	currentTaskType  config.TaskType
	currentTask      config.Task
	sessionSummary   summary.SessionSummary
	isShortSession   bool
	longBreak        config.LongBreak
	cyclePosition    int             // for long break tracking
	commandsWg       *sync.WaitGroup // post commands wg
	commandsCancel   context.CancelFunc

	// ASCII art
	useTimerArt     bool
	timerFont       ascii.Font
	asciiTimerStyle lipgloss.Style

	// database & persistent task
	repo       *db.SessionRepo
	taskRepo   *db.TaskRepo
	activeTask *db.Task
}

func NewModel(taskType config.TaskType, cfg config.Config) Model {
	task := taskType.GetTask()

	var timerFont ascii.Font
	timerStyle := lipgloss.NewStyle()

	if cfg.ASCIIArt.Enabled {
		timerFont = ascii.GetFont(cfg.ASCIIArt.Font)

		timerColor := colors.GetColor(cfg.ASCIIArt.Color)
		timerStyle = timerStyle.Foreground(timerColor)
	}

	sessionSummary := summary.SessionSummary{}

	database, err := db.Connect()
	var repo *db.SessionRepo
	var taskRepo *db.TaskRepo

	if err != nil {
		log.Printf("failed to initialize database: %v", err)
		sessionSummary.SetDatabaseUnavailable()
	} else {
		repo = db.NewSessionRepo(database)
		taskRepo = db.NewTaskRepo(database)
	}

	return Model{
		progressBar:   progress.New(progress.WithDefaultGradient()),
		confirmDialog: confirm.New(),
		help:          help.New(),

		timer:    timer.New(task.Duration),
		duration: task.Duration,

		onSessionEnd:    cfg.OnSessionEnd,
		sessionState:    Running,
		currentTaskType: taskType,
		currentTask:     *task,
		sessionSummary:  sessionSummary,
		longBreak:       cfg.LongBreak,
		cyclePosition:   1,

		useTimerArt:     cfg.ASCIIArt.Enabled,
		timerFont:       timerFont,
		asciiTimerStyle: timerStyle,

		repo:     repo,
		taskRepo: taskRepo,
	}
}

type SessionState byte

const (
	Running SessionState = iota
	Paused
	ShowingConfirm
	WaitingForCommands // waiting for post commands before quitting
	ShowingTaskPicker
	ShowingTaskForm
	Quitting
)

func (m Model) GetSessionSummary() summary.SessionSummary {
	return m.sessionSummary
}

func (m *Model) SetInitialTask(task *db.Task) {
	m.activeTask = task
}

func (m Model) ActiveTask() *db.Task {
	return m.activeTask
}

func (m Model) State() SessionState {
	return m.sessionState
}

func NewModelWithDB(taskType config.TaskType, cfg config.Config, database *sqlx.DB) Model {
	m := NewModel(taskType, cfg)
	if database != nil {
		m.repo = db.NewSessionRepo(database)
		m.taskRepo = db.NewTaskRepo(database)
	}
	return m
}
