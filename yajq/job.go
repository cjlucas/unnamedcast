package yajq

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Job struct {
	ID             int
	Delayed        bool
	DelayedUntil   time.Time
	done           bool
	CreationTime   time.Time
	CompletionTime time.Time
	Priority       int
	Payload        interface{}
	Progress       int
	Logs           []string
}

func (j *Job) asHash() map[string]string {
	hash := map[string]string{
		"id":              strconv.Itoa(int(j.ID)),
		"delayed":         strconv.Itoa(btoi(j.Delayed)),
		"delayed_until":   strconv.Itoa(int(j.DelayedUntil.Unix())),
		"done":            strconv.Itoa(btoi(j.done)),
		"creation_time":   strconv.Itoa(int(j.CreationTime.Unix())),
		"completion_time": strconv.Itoa(int(j.CompletionTime.Unix())),
		"priority":        strconv.Itoa(int(j.Priority)),
	}

	if jsonPayload, err := json.Marshal(&j.Payload); err == nil {
		hash["payload"] = string(jsonPayload)
	}

	return hash
}

type jobUnmarshaller struct {
	Err error
}

func (u *jobUnmarshaller) atoi(s string) int {
	if u.Err != nil || s == "" {
		return 0
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) atob(s string) bool {
	if u.Err != nil || s == "" {
		return false
	}

	val, err := strconv.ParseBool(s)
	if err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) parseJSON(s string) interface{} {
	if u.Err != nil || s == "" {
		return nil
	}

	var val interface{}
	if err := json.Unmarshal([]byte(s), &val); err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) atot(s string) time.Time {
	secs := u.atoi(s)
	if u.Err != nil {
		return time.Time{}
	}

	return time.Unix(int64(secs), 0).UTC()
}

func unmarshalJob(c Conn, key string) (*Job, error) {
	propMap := make(map[string]string)
	jobProps := []string{
		"id",
		"delayed",
		"delayed_until",
		"done",
		"creation_time",
		"priority",
		"payload",
	}

	for _, prop := range jobProps {
		s, err := c.HGet(key, prop)
		if err != nil {
			return nil, err
		}

		propMap[prop] = s
	}

	u := jobUnmarshaller{}
	job := Job{
		ID:             u.atoi(propMap["id"]),
		Delayed:        u.atob(propMap["delayed"]),
		DelayedUntil:   u.atot(propMap["delayed_until"]),
		done:           u.atob(propMap["done"]),
		CreationTime:   u.atot(propMap["creation_time"]),
		CompletionTime: u.atot(propMap["completion_time"]),
		Priority:       u.atoi(propMap["priority"]),
		Payload:        u.parseJSON(propMap["payload"]),
	}

	if u.Err != nil {
		panic(fmt.Sprintf("error during unmarshalling: %s", u.Err))
	}

	return &job, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
