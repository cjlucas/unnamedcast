package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

const SyncTokenVersion = 1

type SyncToken struct {
	Version      int   `json:"version"`
	SyncTimeUnix int64 `json:"sync_time"`
}

func GenerateSyncToken() string {
	return SyncToken{
		Version:      SyncTokenVersion,
		SyncTimeUnix: time.Now().UTC().Unix(),
	}.Encode()
}

func (t SyncToken) SyncTime() time.Time {
	return time.Unix(t.SyncTimeUnix, 0).UTC()
}

func (t SyncToken) Encode() string {
	b, err := json.Marshal(&t)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)
}

func DecodeSyncToken(s string) (SyncToken, error) {
	var t SyncToken
	if s == "" {
		return t, errors.New("No token provided")
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(b, &t)
	return t, err
}
