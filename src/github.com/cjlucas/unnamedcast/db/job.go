package db

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type JobLogEntry struct {
	Time time.Time `json:"time" bson:"time"`
	Line string    `json:"line" bson:"line"`
}

type Job struct {
	ID       ID          `json:"id" bson:"_id,omitempty"`
	KodaID   int         `json:"koda_id" bson:"koda_id" index:",unique"`
	Priority int         `json:"priority" bson:"priority"`
	Queue    string      `json:"queue" bson:"queue" index:"queue"`
	State    string      `json:"state" bson:"state" index:"state"`
	Payload  interface{} `json:"payload" bson:"payload"`
	// CreationTime is the time at which the job was created in koda
	CreationTime     time.Time     `json:"creation_time" bson:"creation_time" index:"creation_time"`
	ModificationTime time.Time     `json:"modification_time" bson:"modification_time"`
	CompletionTime   time.Time     `json:"completion_time" bson:"completion_time" index:"completion_time"`
	Log              []JobLogEntry `json:"log" bson:"log"`
}

type JobCollection struct {
	collection
}

func (c JobCollection) FindByKodaID(id int) *Result {
	return c.Find(&Query{
		Filter: M{"koda_id": id},
	})
}

func (c JobCollection) Create(job Job) (Job, error) {
	job.ID = NewID()
	job.CreationTime = time.Now().UTC()
	job.ModificationTime = job.CreationTime
	return job, c.c.Insert(job)
}

func (c JobCollection) UpdateState(jobID ID, state string) error {
	update := bson.M{"state": state}
	update["modification_time"] = time.Now().UTC()
	if state == "finished" || state == "dead" {
		update["completion_time"] = update["modification_time"]
	}
	return c.c.UpdateId(jobID, bson.M{"$set": update})
}

func (c JobCollection) AppendLog(jobID ID, line string) error {
	entry := JobLogEntry{
		Time: time.Now().UTC(),
		Line: line,
	}
	return c.c.Update(bson.M{"_id": jobID}, bson.M{
		"$push": bson.M{"log": entry},
	})
}
