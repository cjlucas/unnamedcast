package utctime

import (
	"encoding/json"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(text []byte) error {
	err := json.Unmarshal(text, &t.Time)
	if err != nil {
		return err
	}

	t.Time = t.UTC()
	return nil
}

func (t Time) Add(dur time.Duration) Time {
	return Time{t.Time.Add(dur)}
}

func (t *Time) Before(time Time) bool {
	return t.Time.UTC().Before(time.Time.UTC())
}

func (t Time) Equal(time Time) bool {
	return t.UTC() == time.UTC()
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC())
}

// When mgo.v2/bson checks if a concrete type is a Getter, it does not check if its
// pointer type implements it, so we must implement Getter on the concrete type
//
// Also, implementing Setter is unncessary as mgo.v2/bson will unmarshal it as
// UTC for us

func (t Time) GetBSON() (interface{}, error) {
	return t.UTC(), nil
}

func (t *Time) SetBSON(raw bson.Raw) error {
	err := raw.Unmarshal(&t.Time)
	if err != nil {
		return err
	}

	t.Time = t.UTC()
	return nil
}

func Now() Time {
	return Time{time.Now().UTC()}
}
