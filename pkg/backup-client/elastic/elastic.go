package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const snapshotEndpoint = "_snapshot"

type Client struct {
	baseURL              string
	httpClient           *http.Client
	backupRepositoryName string
}

func NewElasticClient(baseUrl, repositoryName string) *Client {
	return &Client{
		baseURL: baseUrl,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		backupRepositoryName: repositoryName,
	}
}

func (e Client) elasticRequestPath(id int64) string {
	snapshotName := fmt.Sprintf("%s-%d", "camunda_zeebe_records", id)
	requestPath := fmt.Sprintf("http://%s/%s/%s/%s", e.baseURL, snapshotEndpoint, e.backupRepositoryName, snapshotName)
	return requestPath
}

func (e Client) GetBackup(ctx context.Context, id int64) (*SnapshotResponse, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.elasticRequestPath(id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var snapshotResponse SnapshotResponse
		err = json.Unmarshal(respBody, &snapshotResponse)
		if err != nil {
			return nil, err
		}
		// Todo: check the state.
		return &snapshotResponse, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Ignore not found error, as we will trigger a snapshot then.
		// ToDo: Check what is returned when a backup repository is not configured
		return nil, nil
	}

	return nil, fmt.Errorf("error getting the elastic snapshot")
}

func (e Client) RequestSnapshot(ctx context.Context, id int64, zeebeIndexPrefix string) (*SnapshotResponse, error) {
	requestBody := []byte(fmt.Sprintf(`{"indices": "%s","feature_states": ["none"]}`, zeebeIndexPrefix))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, e.elasticRequestPath(id), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	// Todo: Check response more
	log.Printf("Elastic started snapshot of %s and response: response Body %s\n", requestBody, string(respBody))

	switch resp.StatusCode {
	case http.StatusOK:
		return &SnapshotResponse{}, nil
	case http.StatusNotFound:
		return nil, errors.New(string(respBody))
	}

	return nil, err
}

// elasticSnapshotZeebeRecords first tries to get information about the backup and returns the information. If there is no information
// about a backup it requests a backup

func (e Client) DeleteSnapshot(ctx context.Context, id int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, e.elasticRequestPath(id), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}

	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	// Todo: Check response more
	fmt.Println("response Body : ", string(respBody))
	// respBody should be
	//{
	//	"acknowledged" : true
	//}
	// Todo: Check if deletion was acknowledged
	return nil
}
