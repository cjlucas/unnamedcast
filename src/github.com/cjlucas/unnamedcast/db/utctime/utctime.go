package utctime

import (
	"encoding/json"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Time struct {
	t time.Time
}

func (t *Time) UnmarshalJSON(text []byte) error {
	err := json.Unmarshal(text, &t.t)
	if err != nil {
		return err
	}

	t.t = t.t.UTC()
	return nil
}

func (t *Time) IsZero() bool {
	return t.t.IsZero()
}

func (t *Time) Format(layout string) string {
	return t.t.UTC().Format(layout)
}

func (t *Time) Add(dur time.Duration) Time {
	return Time{t.t.Add(dur)}
}

func (t *Time) Before(time Time) bool {
	return t.t.UTC().Before(time.t.UTC())
}

func (t Time) Equal(time Time) bool {
	return t.t.UTC() == time.t.UTC()
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.t.UTC())
}

// When mgo.v2/bson checks if a concrete type is a Getter, it does not check if its
// pointer type implements it, so we must implement Getter on the concrete type
//
// Also, implementing Setter is unncessary as mgo.v2/bson will unmarshal it as
// UTC for us

func (t Time) GetBSON() (interface{}, error) {
	return t.t.UTC(), nil
}

func (t *Time) SetBSON(raw bson.Raw) error {
	err := raw.Unmarshal(&t.t)
	if err != nil {
		return err
	}

	t.t = t.t.UTC()
	return nil
}

func Now() Time {
	return Time{time.Now().UTC()}
}
