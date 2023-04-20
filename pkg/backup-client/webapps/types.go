package webapps

import "time"

// Optimize:
//
//	{
//		"errorCode" : "notFoundError",
//		"errorMessage" : "The server could not find the requested resource.",
//		"detailedMessage" : "No Optimize backup with ID [1679426843] could be found.",
//		"reportDefinition" : null
//	}
//
// Operate & Tasklist
//
//	{
//		"timestamp" : "2023-03-21T20:50:28.707+00:00",
//		"status" : 404,
//		"error" : "Not Found",
//		"message" : "No message available",
//		"path" : "/actuator/backups/1679431820"
//	}
type ErrorBackupResponse struct {
	// Optimize
	ErrorCode       string `json:"errorCode"`
	ErrorMessage    string `json:"errorMessage"`
	DetailedMessage string `json:"detailedMessage"`

	// Operate and Tasklist
	Timestamp time.Time `json:"timestamp"`
	Status    int       `json:"status"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Path      string    `json:"path"`
}

type BackupResponse struct {
	// Operate has int64 backupID
	// Optimize has int64 backupID
	// Tasklist has string backupID // To be fixed - soon TM
	BackupId      int64  `json:"backupId"`
	State         string `json:"state"`
	FailureReason string `json:"failureReason"`
	Details       []struct {
		SnapshotName string   `json:"snapshotName"`
		State        string   `json:"state"`
		StartTime    string   `json:"startTime"`
		Failures     []string `json:"failures"`
	} `json:"details"`
}

type backupRequestBody struct {
	BackupID string `json:"backupId"`
}

type backupRequestErrorMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Status    int       `json:"status"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Path      string    `json:"path"`
}
