package webapps

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	OperateApp  = "operate"
	OptimizeApp = "optimize"
	TasklistApp = "tasklist"
)

type BackupClient struct {
	name       string
	baseURL    string
	httpClient *http.Client
}

func NewBackupClient(name string, baseURL string) (*BackupClient, error) {
	switch name {
	case OptimizeApp, OperateApp, TasklistApp:
		url := fmt.Sprintf("http://%s/actuator/backups", baseURL)
		return &BackupClient{
			name:    name,
			baseURL: url,
			httpClient: &http.Client{
				Timeout: time.Second * 10,
			},
		}, nil
	default:
		return nil, errors.New("application not supported")
	}
}

func (b BackupClient) Name() string {
	return b.name
}
