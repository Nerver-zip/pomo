package cmd

import (
	"github.com/Bahaaio/pomo/ui/stats"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Args:  cobra.MaximumNArgs(0),
	Short: "Display Pomodoro statistics and productivity metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, _ := cmd.Flags().GetInt("task")
		if taskID > 0 {
			repo, cleanup, err := getTaskRepo()
			if err != nil {
				return err
			}
			defer cleanup()

			taskStats, err := repo.GetTaskStats(taskID)
			if err != nil {
				return err
			}

			cmd.Printf("Task #%d: %s\n", taskStats.TaskID, taskStats.Title)
			cmd.Printf("  Status:       %s\n", taskStats.Status)
			cmd.Printf("  Created:      %s\n", taskStats.CreatedAt.Format("2006-01-02 15:04"))
			if taskStats.CompletedAt != nil {
				cmd.Printf("  Completed:    %s\n", taskStats.CompletedAt.Format("2006-01-02 15:04"))
			}
			cmd.Printf("  Pomodoros:    %d completed\n", taskStats.TotalPomodoros)
			cmd.Printf("  Total Focus:  %s\n", formatDuration(taskStats.TotalDuration))
			return nil
		}

		m := stats.New()
		p := tea.NewProgram(m, tea.WithAltScreen())

		_, err := p.Run()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	statsCmd.Flags().IntP("task", "T", 0, "display focus metrics for a specific task ID")
	rootCmd.AddCommand(statsCmd)
}
