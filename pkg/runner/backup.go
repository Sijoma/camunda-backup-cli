package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	wg := sync.WaitGroup{}
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
		// Monitor Webapp Backups
		wg.Add(1)
		go waitUntilBackupCompleted(ctx, &wg, operate)
	}

	// Optimize
	if definition.optimizeURL != "" {
		optimize, _ := webapps.NewBackupClient("optimize", definition.optimizeURL)
		err := optimize.RequestBackup(ctx, backupID)
		if err != nil {
			log.Println("optimize backup request failed")
			log.Fatal(err)
		}
		// Monitor Webapp Backups
		wg.Add(1)
		go waitUntilBackupCompleted(ctx, &wg, optimize)
	}
	// Tasklist
	if definition.tasklistURL != "" {
		tasklist, _ := webapps.NewBackupClient("tasklist", definition.tasklistURL)
		err := tasklist.RequestBackup(ctx, backupID)
		if err != nil {
			log.Println("tasklist backup request failed")
			log.Fatal(err)
		}
		wg.Add(1)
		go waitUntilBackupCompleted(ctx, &wg, tasklist)
	}

	wg.Wait()
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
		wg.Add(1)
		go waitUntilZeebeBackupCompleted(ctx, &wg, zeebe)

	}

	if definition.elasticURL != "" {
		elasticBkp := elastic.NewElasticClient(definition.elasticURL, definition.backupRepositoryName)
		_, err := elasticBkp.RequestSnapshot(ctx, backupID, definition.zeebeIndexPrefix)
		if err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go waitUntilElasticCompleted(ctx, &wg, elasticBkp)
	}

	// Wait for Zeebe and/or elastic
	wg.Wait()
	log.Println("✅ ✅ ✅ ELASTIC and ZEEBE DONE ✅ ✅ ✅")
	log.Println("🚀🚀🚀backup DONE!🚀🚀🚀")
	log.Println("BackupID: ", backupID)
}

func waitUntilBackupCompleted(ctx context.Context, wg *sync.WaitGroup, client *webapps.BackupClient) {
	defer wg.Done()
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
			log.Println("waiting on ", client.Name(), pollInterval)
			time.Sleep(pollInterval)
		}
	}()

	select {
	case res := <-completedBackup:
		log.Printf("✅ %s Done! %s %s\n", client.Name(), res.State, res.FailureReason)
		return
	case <-time.After(timeout):
		log.Printf("%s timed out\n", client.Name())
		return
	}
}

func waitUntilElasticCompleted(ctx context.Context, wg *sync.WaitGroup, client *elastic.Client) {
	defer wg.Done()
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

	select {
	case res := <-completedBackup:
		for _, snapshot := range res.Snapshots {
			log.Printf("Elastic Snapshot in state %s. Name: %s", snapshot.State, snapshot.Snapshot)
		}
		//bkpJson, err := json.MarshalIndent(res, "", "  ")
		//if err != nil {
		//	log.Fatalf(err.Error())
		//}
		//fmt.Printf("Elastic Completed %s\n", string(bkpJson))
		return
	case <-time.After(timeout):
		log.Println("elastic snapshot timed out")
		return
	}
}

func waitUntilZeebeBackupCompleted(ctx context.Context, wg *sync.WaitGroup, client *zeebeBackup.BackupClient) {
	defer wg.Done()
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

	select {
	case res := <-completedBackup:
		log.Printf("✅ %s Done! %s \n", "zeebe", res.State)
		return
	case <-time.After(timeout):
		log.Printf("%s timed out\n", "zeebe")
		return
	}
}
