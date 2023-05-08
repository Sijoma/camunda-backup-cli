/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"c8backup/pkg/restore"
	"github.com/spf13/cobra"
)

var backupID int64
var namespace string

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("restore called")
		// Todo: Refactor this with something easier :)
		restore.Restore(namespace, backupID, elasticURL, operateURL, tasklistURL, optimizeURL, zeebeURL, elasticSnapshotRepositoryName)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVar(&namespace, "namespace", "", "namespace where stuff runs")
	restoreCmd.Flags().Int64Var(&backupID, "backupID", 0, "ID of the the backup to restore")
	restoreCmd.Flags().StringVar(&elasticURL, "elastic", "", "Pass in the url to the elastic mgmt endpoint")
	restoreCmd.Flags().StringVar(&elasticSnapshotRepositoryName, "elastic-repository", "", "Name of the elasticsearch snapshot repository")
	restoreCmd.Flags().StringVar(&operateURL, "operate", "", "Pass in the url to the operate mgmt endpoint")
	restoreCmd.Flags().StringVar(&tasklistURL, "tasklist", "", "Pass in the url to the tasklist mgmt endpoint")
	restoreCmd.Flags().StringVar(&optimizeURL, "optimize", "", "Pass in the url to the optimize mgmt endpoint")

	restoreCmd.Flags().StringVar(&zeebeURL, "zeebe", "", "Pass in the url to the zeebe mgmt endpoint")
}
