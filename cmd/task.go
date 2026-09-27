package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/db"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"tasks"},
	Short:   "Manage persistent tasks for pomodoro sessions",
	Long: `Manage persistent tasks to track focused Pomodoro time.

Tasks give context to your focus sessions. Use 'pomo task add' to create tasks,
'pomo task list' to view them, and 'pomo --task <id>' or 'pomo task start <id>' to focus on them.`,
}

var taskAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		desc, _ := cmd.Flags().GetString("description")

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.Create(title, desc)
		if err != nil {
			return err
		}

		cmd.Printf("Created task #%d: %q\n", task.ID, task.Title)
		return nil
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		doneOnly, _ := cmd.Flags().GetBool("done")

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		tasks, err := repo.ListAll(all || doneOnly)
		if err != nil {
			return err
		}

		if doneOnly {
			var filtered []db.Task
			for _, t := range tasks {
				if t.Status == db.TaskCompleted {
					filtered = append(filtered, t)
				}
			}
			tasks = filtered
		}

		if len(tasks) == 0 {
			if doneOnly {
				cmd.Println("No completed tasks found.")
			} else {
				cmd.Println("No pending tasks. Run 'pomo task add <title>' to create one.")
			}
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tPOMODOROS\tFOCUSED\tTITLE")

		for _, t := range tasks {
			pomodorosStr := strconv.Itoa(t.TotalPomodoros)
			focusedStr := formatDuration(t.TotalDuration)
			fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\n", t.ID, t.Status, pomodorosStr, focusedStr, t.Title)
		}
		return w.Flush()
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task's title or description",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")

		if title == "" {
			title = task.Title
		}
		if !cmd.Flags().Changed("description") {
			desc = task.Description
		}

		if err := repo.Update(id, title, desc); err != nil {
			return err
		}

		cmd.Printf("Updated task #%d: %q\n", id, title)
		return nil
	},
}

var taskDoneCmd = &cobra.Command{
	Use:     "done <id>",
	Aliases: []string{"complete"},
	Short:   "Mark a task as completed",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		if err := repo.Complete(id); err != nil {
			return err
		}

		cmd.Printf("Completed task #%d: %q 🎉\n", id, task.Title)
		return nil
	},
}

var taskReopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a completed task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		if err := repo.Reopen(id); err != nil {
			return err
		}

		cmd.Printf("Reopened task #%d: %q\n", id, task.Title)
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		if err := repo.Delete(id); err != nil {
			return err
		}

		cmd.Printf("Deleted task #%d: %q\n", id, task.Title)
		return nil
	},
}

var taskStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start a Pomodoro timer with an attached task",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Without ID: start normal work session (user can press 't' inside TUI)
			runTask(config.WorkTask, cmd)
			return nil
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		task, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		// Inject task ID into root flag and run
		_ = cmd.Flags().Set("task", strconv.Itoa(task.ID))
		runTask(config.WorkTask, cmd)
		return nil
	},
}

var taskStatsCmd = &cobra.Command{
	Use:   "stats <id>",
	Short: "Display focus metrics for a specific task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID %q: must be a number", args[0])
		}

		repo, cleanup, err := getTaskRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		stats, err := repo.GetTaskStats(id)
		if err != nil {
			return err
		}

		cmd.Printf("Task #%d: %s\n", stats.TaskID, stats.Title)
		cmd.Printf("  Status:       %s\n", stats.Status)
		cmd.Printf("  Created:      %s\n", stats.CreatedAt.Format("2006-01-02 15:04"))
		if stats.CompletedAt != nil {
			cmd.Printf("  Completed:    %s\n", stats.CompletedAt.Format("2006-01-02 15:04"))
		}
		cmd.Printf("  Pomodoros:    %d completed\n", stats.TotalPomodoros)
		cmd.Printf("  Total Focus:  %s\n", formatDuration(stats.TotalDuration))
		return nil
	},
}

func getTaskRepo() (*db.TaskRepo, func(), error) {
	database, err := db.Connect()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db.NewTaskRepo(database), func() {
		_ = database.Close()
	}, nil
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0m"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func init() {
	// Flags for task add
	taskAddCmd.Flags().StringP("description", "d", "", "task description")

	// Flags for task list
	taskListCmd.Flags().BoolP("all", "a", false, "include completed tasks")
	taskListCmd.Flags().BoolP("done", "d", false, "show only completed tasks")

	// Flags for task edit
	taskEditCmd.Flags().StringP("title", "t", "", "new task title")
	taskEditCmd.Flags().StringP("description", "d", "", "new task description")

	// Flags for task delete
	taskDeleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	// Flags for task start
	taskStartCmd.Flags().IntP("task", "T", 0, "task ID to focus on")

	// Register subcommands
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskStatsCmd)

	// Add taskCmd to rootCmd
	rootCmd.AddCommand(taskCmd)
}
