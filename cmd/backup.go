/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"c8backup/pkg/runner"
	"github.com/spf13/cobra"
)

var zeebeURL string
var operateURL string
var tasklistURL string
var optimizeURL string
var elasticURL string
var elasticSnapshotRepositoryName string

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
			// Todo: Make zeebe-record prefix configurable
			Elastic(elasticURL, elasticSnapshotRepositoryName).
			Zeebe(zeebeURL).
			Build()

		runner.DoBackup(backupDefinition)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVar(&zeebeURL, "zeebe", "", "Pass in the url to the zeebe mgmt endpoint")
	backupCmd.Flags().StringVar(&operateURL, "operate", "", "Pass in the url to the operate mgmt endpoint")
	backupCmd.Flags().StringVar(&tasklistURL, "tasklist", "", "Pass in the url to the tasklist mgmt endpoint")
	backupCmd.Flags().StringVar(&optimizeURL, "optimize", "", "Pass in the url to the optimize mgmt endpoint")

	backupCmd.Flags().StringVar(&elasticURL, "elastic", "", "Pass in the url to the elastic mgmt endpoint")
	backupCmd.Flags().StringVar(&elasticSnapshotRepositoryName, "repository", "", "Name of the elasticsearch snapshot repository")
	backupCmd.MarkFlagsRequiredTogether("elastic", "repository")
}
