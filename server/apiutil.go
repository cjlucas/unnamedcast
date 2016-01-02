package main

import (
	"encoding/json"
	"time"
)

const SyncTokenVersion = 1

type SyncToken struct {
	Version  int   `json:"version"`
	SyncTime int64 `json:"sync_time"`
}

func GenerateSyncToken() string {
	payload, _ := json.Marshal(SyncToken{
		Version:  SyncTokenVersion,
		SyncTime: time.Now().UTC().Unix(),
	})

	return string(payload)
}

func ParseSyncToken(data []byte) (SyncToken, error) {
	var token SyncToken
	err := json.Unmarshal(data, &token)
	return token, err
}
