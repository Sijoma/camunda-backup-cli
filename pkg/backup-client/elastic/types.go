package elastic

import "time"

type SnapshotResponse struct {
	Snapshots []struct {
		Snapshot           string        `json:"snapshot"`
		Uuid               string        `json:"uuid"`
		Repository         string        `json:"repository"`
		VersionId          int           `json:"version_id"`
		Version            string        `json:"version"`
		Indices            []interface{} `json:"indices"`
		DataStreams        []interface{} `json:"data_streams"`
		IncludeGlobalState bool          `json:"include_global_state"`
		State              string        `json:"state"`
		StartTime          time.Time     `json:"start_time"`
		StartTimeInMillis  int64         `json:"start_time_in_millis"`
		EndTime            time.Time     `json:"end_time"`
		EndTimeInMillis    int64         `json:"end_time_in_millis"`
		DurationInMillis   int           `json:"duration_in_millis"`
		Failures           []interface{} `json:"failures"`
		Shards             struct {
			Total      int `json:"total"`
			Failed     int `json:"failed"`
			Successful int `json:"successful"`
		} `json:"shards"`
		FeatureStates []interface{} `json:"feature_states"`
	} `json:"snapshots"`
	Total     int `json:"total"`
	Remaining int `json:"remaining"`
}
