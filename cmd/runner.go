package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
	"github.com/Bahaaio/pomo/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func runTask(taskType config.TaskType, cmd *cobra.Command) {
	task := taskType.GetTask()

	if err := parseArguments(cmd.Flags().Args(), task, &config.C.Break); err != nil {
		_ = cmd.Usage()
		die(err)
	}

	if err := parseFlags(cmd, &config.C.Work); err != nil {
		die(err)
	}

	log.Printf("starting %v session: %v", taskType.GetTask().Title, taskType.GetTask().Duration)

	m := ui.NewModel(taskType, config.C)
	if cmd.Flags().Lookup("task") != nil {
		if taskID, err := cmd.Flags().GetInt("task"); err == nil && taskID > 0 {
			database, err := db.Connect()
			if err != nil {
				die(fmt.Errorf("failed to connect to database: %w", err))
			}
			taskRepo := db.NewTaskRepo(database)
			t, err := taskRepo.GetByID(taskID)
			if err != nil {
				die(fmt.Errorf("task #%d not found: %w", taskID, err))
			}
			m.SetInitialTask(t)
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		die(err)
	}

	// print session summary
	finalModel.(ui.Model).GetSessionSummary().Print()
}

// parses the arguments and sets the duration
// returns an error if the duration is invalid
func parseArguments(args []string, task *config.Task, breakTask *config.Task) error {
	if len(args) > 0 {
		var err error
		task.Duration, err = time.ParseDuration(args[0])
		if err != nil {
			return fmt.Errorf("invalid duration: '%v'", args[0])
		}

		if len(args) > 1 {
			breakTask.Duration, err = time.ParseDuration(args[1])
			if err != nil {
				return fmt.Errorf("invalid break duration: '%v'", args[1])
			}
		}
	}

	return nil
}

// parses the flags and sets the title
func parseFlags(cmd *cobra.Command, workTask *config.Task) error {
	title, _ := cmd.Flags().GetString("title")

	// discard empty title
	if title != "" {
		workTask.Title = title
	}

	return nil
}
