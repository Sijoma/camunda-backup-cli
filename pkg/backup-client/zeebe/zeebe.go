package zeebeBackup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type action string

type BackupClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewZeebeClient(baseURL string) *BackupClient {
	url := fmt.Sprintf("http://%s/", baseURL)

	return &BackupClient{
		baseURL: url,
		httpClient: &http.Client{
			// We use a short timeout to not block reconcile loop
			Timeout: time.Second * 10,
		},
	}
}

func (z BackupClient) ResumeExporting(ctx context.Context) error {
	return z.exportingRequest(ctx, "resume")
}

func (z BackupClient) StopExporting(ctx context.Context) error {
	return z.exportingRequest(ctx, "pause")
}

func (z BackupClient) exportingRequest(ctx context.Context, action action) error {
	path := "actuator/exporting"
	requestPath := fmt.Sprintf("%s%s/%s", z.baseURL, path, action)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestPath, nil)
	if err != nil {
		return err
	}

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("zeebe request to %s exporting failed", string(action))
	}
	return err
}

func (z BackupClient) GetBackup(ctx context.Context, id int64) (*BackupResponse, error) {
	requestPath := fmt.Sprintf("%sactuator/backups/%d", z.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Ignore not found, then we return nothing and start a backup
	if resp.StatusCode == http.StatusNotFound {
		// ToDo: Check cases if there is a 404 that means something else
		return nil, nil
	}

	if resp.StatusCode >= 300 {
		// There is an error other than not found ... lets retry
		// Todo: Check to fail completely on configuration errors
		var backupErrorBody BackupMessageResponse
		err = json.Unmarshal(respBody, &backupErrorBody)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("there was an error %s", backupErrorBody.Message)
	}

	var backupResp BackupResponse
	err = json.Unmarshal(respBody, &backupResp)
	if err != nil {
		return nil, err
	}

	return &backupResp, err
}

func (z BackupClient) RequestBackup(ctx context.Context, id int64) error {
	requestPath := fmt.Sprintf("%sactuator/backups", z.baseURL)
	body := BackupRequestBody{BackupID: id}
	backupRequestJson, _ := json.Marshal(body)
	requestBody := bytes.NewBuffer(backupRequestJson)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestPath, requestBody)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		var backupErrorBody BackupMessageResponse
		err = json.Unmarshal(respBody, &backupErrorBody)
		if err != nil {
			return err
		}
		return fmt.Errorf("backup Request failed: %s", backupErrorBody.Message)
	}

	// Todo: Probably remove, the only useful information is really the status code.
	// We could check the message though but that's not necessary
	var backupResp BackupMessageResponse
	err = json.Unmarshal(respBody, &backupResp)
	if err != nil {
		return err
	}

	// The backup was accepted we return info about it
	return nil
}

func (z BackupClient) DeleteBackup(ctx context.Context, id int64) error {
	requestPath := fmt.Sprintf("%s/%d", z.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestPath, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		return err
	default:
		//400 Bad Request	There is an issue with the request, for example the repository name specified in the Optimize configuration does not exist. Refer to returned error message for details.
		//500 Server Error	An error occurred, for example the snapshot repository does not exist. Refer to the returned error message for details.
		//502 Bad Gateway	Optimize has encountered issues while trying to connect to Elasticsearch.
		return fmt.Errorf("DeleteBackup: error deleting backup")
	}
}
