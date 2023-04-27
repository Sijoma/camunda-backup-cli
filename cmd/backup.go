package cmd

import (
	"fmt"
	"os"

	"c8backup/internal/tui"
	"c8backup/pkg/runner"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	zeebeURL                      string
	zeebeIndexPrefix              string
	operateURL                    string
	tasklistURL                   string
	optimizeURL                   string
	elasticURL                    string
	elasticSnapshotRepositoryName string
	verbose                       bool
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "backup C8 platform",
	Long:  `Backup Camunda 8 Platform`,
	Run: func(cmd *cobra.Command, args []string) {
		backupDefinition := runner.NewBackupDefinitionBuilder().
			Operate(operateURL).
			Tasklist(tasklistURL).
			Optimize(optimizeURL).
			Elastic(elasticURL, elasticSnapshotRepositoryName).
			Zeebe(zeebeURL).
			ZeebeIndexPrefix(zeebeIndexPrefix).
			Build()

		statusReporter := make(chan runner.BackupDefinition, 1)
		m := tui.Model{
			Progress: progress.New(progress.WithDefaultGradient()),
		}

		p := tea.NewProgram(m)
		go listenToStatus(p, statusReporter)
		go runner.DoBackup(backupDefinition, statusReporter)

		if _, err := p.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVar(&elasticURL, "elastic", "", "Pass in the url to the elastic mgmt endpoint")
	backupCmd.Flags().StringVar(&elasticSnapshotRepositoryName, "elastic-repository", "", "Name of the elasticsearch snapshot repository")
	backupCmd.Flags().StringVar(&operateURL, "operate", "", "Pass in the url to the operate mgmt endpoint")
	backupCmd.Flags().StringVar(&tasklistURL, "tasklist", "", "Pass in the url to the tasklist mgmt endpoint")
	backupCmd.Flags().StringVar(&optimizeURL, "optimize", "", "Pass in the url to the optimize mgmt endpoint")
	backupCmd.Flags().StringVar(&zeebeURL, "zeebe", "", "Pass in the url to the zeebe mgmt endpoint")
	backupCmd.Flags().StringVar(&zeebeIndexPrefix, "zeebe-index-prefix", "zeebe-record*", "Pass in the zeebe elasticsearch record prefix. Default: 'zeebe-record*'")
	backupCmd.Flags().BoolVar(&verbose, "verbose", false, "Show log output")
	backupCmd.MarkFlagsRequiredTogether("elastic", "elastic-repository")
}

func listenToStatus(p *tea.Program, status <-chan runner.BackupDefinition) {
	for {
		select {
		case message := <-status:
			p.Send(message)
		}
	}

}
