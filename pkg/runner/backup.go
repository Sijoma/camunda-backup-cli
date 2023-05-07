package runner

import (
	"context"
	"fmt"
	"log"
	"time"

	"c8backup/pkg/backup-client/elastic"
	"c8backup/pkg/backup-client/webapps"
	"c8backup/pkg/backup-client/zeebe"
)

var backupID int64

type BackupDefinition struct {
	elasticURL           string
	operateURL           string
	tasklistURL          string
	optimizeURL          string
	zeebeURL             string
	zeebeIndexPrefix     string
	backupID             int64
	backupRepositoryName string
}

const timeout = time.Minute
const pollInterval = time.Second * 5

func DoBackup(definition BackupDefinition) {
	ctx := context.Background()
	backupID = definition.backupID

	// Operate
	if definition.operateURL != "" {
		operate, _ := webapps.NewBackupClient("operate", definition.operateURL)
		err := operate.RequestBackup(ctx, backupID)
		if err != nil {
			log.Println("operate backup request failed")
			log.Fatal(err)
		}
		operateBackup := pollUntilBackupCompleted(ctx, operate)
		handleResponse(operateBackup, operate.Name())
	}
	// Optimize
	if definition.optimizeURL != "" {
		optimize, _ := webapps.NewBackupClient("optimize", definition.optimizeURL)
		err := optimize.RequestBackup(ctx, backupID)
		if err != nil {
			log.Println("optimize backup request failed")
			log.Fatal(err)
		}
		optimizeBackup := pollUntilBackupCompleted(ctx, optimize)
		handleResponse(optimizeBackup, optimize.Name())
	}
	// Tasklist
	if definition.tasklistURL != "" {
		tasklist, _ := webapps.NewBackupClient("tasklist", definition.tasklistURL)
		err := tasklist.RequestBackup(ctx, backupID)
		if err != nil {
			log.Println("tasklist backup request failed")
			log.Fatal(err)
		}
		tasklistBackup := pollUntilBackupCompleted(ctx, tasklist)
		handleResponse(tasklistBackup, tasklist.Name())
	}
	log.Println("✅ ✅ ✅ WEBAPPS  ✅ ✅ ✅")

	// Once Webapps are finished
	if definition.zeebeURL != "" {
		zeebe := zeebeBackup.NewZeebeClient(definition.zeebeURL)
		defer func(zeebe *zeebeBackup.BackupClient, ctx context.Context) {
			err := zeebe.ResumeExporting(ctx)
			if err != nil {
				log.Fatal("Error resuming zeebe export", err)
			}
			log.Println("▶️▶️▶️ZEEBE EXPORT RESUMED ▶️▶️▶️")
		}(zeebe, ctx)
		// Zeebe Stop Exporting
		err := zeebe.StopExporting(ctx)
		if err != nil {
			fmt.Println("Error stopping zeebe export ", definition.backupID)
			return
		}
		log.Println("⏸️ ⏸️ ⏸️ ️ZEEBE EXPORT STOPPED  ⏸️ ⏸️ ⏸️")
		err = zeebe.RequestBackup(ctx, backupID)
		if err != nil {
			log.Fatal(err)
		}
		completedBackup := waitUntilZeebeBackupCompleted(ctx, zeebe)
		select {
		case res := <-completedBackup:
			log.Printf("✅ %s Done! %s \n", "zeebe", res.State)
			log.Println("✅ ✅ ✅ ZEEBE DONE")
			return
		case <-time.After(timeout):
			log.Printf("%s timed out\n", "zeebe")
			return
		}

	}

	if definition.elasticURL != "" {
		elasticBkp := elastic.NewElasticClient(definition.elasticURL, definition.backupRepositoryName)
		_, err := elasticBkp.RequestSnapshot(ctx, backupID, definition.zeebeIndexPrefix)
		if err != nil {
			log.Fatal(err)
		}
		completedBackup := pollUntilElasticCompleted(ctx, elasticBkp)
		select {
		case res := <-completedBackup:
			for _, snapshot := range res.Snapshots {
				log.Printf("Elastic Snapshot in state %s. Name: %s", snapshot.State, snapshot.Snapshot)
			}
			log.Println("✅ ✅ ✅ ELASTIC DONE")
			return
		case <-time.After(timeout):
			log.Println("elastic snapshot timed out")
			return
		}
	}
	log.Println("🚀🚀🚀backup DONE!🚀🚀🚀")
	log.Println("BackupID: ", backupID)
}

func handleResponse(completedBackup <-chan webapps.BackupResponse, name string) {
	select {
	case res := <-completedBackup:
		log.Printf("✅ %s Done! %s %s\n", name, res.State, res.FailureReason)
		return
	case <-time.After(timeout):
		log.Printf("%s timed out\n", name)
		return
	}
}

func pollUntilBackupCompleted(ctx context.Context, client *webapps.BackupClient) <-chan webapps.BackupResponse {
	completedBackup := make(chan webapps.BackupResponse, 1)
	go func() {
		for {
			backupInfo, err := client.GetBackup(ctx, backupID)
			if err != nil {
				log.Println(err)
				continue
			}
			if backupInfo != nil {
				log.Printf("%s Backup in state: %s \n", client.Name(), backupInfo.State)
				if backupInfo.State == "COMPLETED" {
					completedBackup <- *backupInfo
					return
				}
			}
			time.Sleep(pollInterval)
		}
	}()

	return completedBackup
}

func pollUntilElasticCompleted(ctx context.Context, client *elastic.Client) <-chan elastic.SnapshotResponse {
	completedBackup := make(chan elastic.SnapshotResponse, 1)
	go func() {
		for {
			backupInfo, err := client.GetBackup(ctx, backupID)
			if err != nil {
				log.Println(err)
				continue
			}
			if backupInfo != nil {
				log.Printf("Elastic in state: %v. \n", backupInfo.Snapshots[0].State)
				if backupInfo.Snapshots[0].State == "SUCCESS" {
					completedBackup <- *backupInfo
					return
				}

			}

			log.Println("waiting elastic", pollInterval)
			time.Sleep(pollInterval)
		}
	}()

	return completedBackup
}

func waitUntilZeebeBackupCompleted(ctx context.Context, client *zeebeBackup.BackupClient) <-chan zeebeBackup.BackupResponse {
	completedBackup := make(chan zeebeBackup.BackupResponse, 1)
	go func() {
		for {
			backupInfo, err := client.GetBackup(ctx, backupID)
			if err != nil {
				log.Println(err)
				continue
			}
			if backupInfo != nil {
				log.Printf("%s Backup in state: %s \n", "zeebe", backupInfo.State)
				if backupInfo.State == "COMPLETED" {
					completedBackup <- *backupInfo
					return
				}

			}
			log.Println("sleeping zeebe", pollInterval)
			time.Sleep(pollInterval)
		}
	}()

	return completedBackup
}
