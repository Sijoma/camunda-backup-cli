package zeebeBackup

import "time"

type BackupRequestBody struct {
	BackupID int64 `json:"backupId"`
}

type BackupMessageResponse struct {
	Message string `json:"message"`
}

type BackupResponse struct {
	BackupId int64  `json:"backupId"`
	State    string `json:"state"`
	Details  []struct {
		PartitionId        int       `json:"partitionId"`
		State              string    `json:"state"`
		CreatedAt          time.Time `json:"createdAt"`
		LastUpdatedAt      time.Time `json:"lastUpdatedAt"`
		SnapshotId         string    `json:"snapshotId"`
		CheckpointPosition int       `json:"checkpointPosition"`
		BrokerVersion      string    `json:"brokerVersion"`
	} `json:"details"`
}
