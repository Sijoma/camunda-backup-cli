package webapps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func (b BackupClient) GetBackup(ctx context.Context, id int64) (*BackupResponse, error) {
	requestPath := fmt.Sprintf("%s/%d", b.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var successBackupResp BackupResponse
	err = json.Unmarshal(respBody, &successBackupResp)
	if err != nil {
		return nil, err
	}

	return &successBackupResp, err
}

func (b BackupClient) RequestBackup(ctx context.Context, id int64) error {
	body := backupRequestBody{BackupID: strconv.FormatInt(id, 10)}
	backupRequestJson, _ := json.Marshal(body)
	requestBody := bytes.NewBuffer(backupRequestJson)
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL, requestBody)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		var backupErrorBody backupRequestErrorMessage
		err = json.Unmarshal(respBody, &backupErrorBody)
		if err != nil {
			return fmt.Errorf("backup Request failed for backupId %d, %w", id, err)
		}

		return fmt.Errorf("backup Request failed: %s, %s, %w", backupErrorBody.Error, backupErrorBody.Message, err)
	}
	return nil
}

func (b BackupClient) DeleteBackup(ctx context.Context, id int64) error {
	requestPath := fmt.Sprintf("%s/%d", b.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestPath, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := b.httpClient.Do(req)
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
		return err
	}
}
