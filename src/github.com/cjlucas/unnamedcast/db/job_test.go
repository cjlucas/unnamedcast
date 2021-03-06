package db

import "testing"

func TestJob_FindByKodaID(t *testing.T) {
	db := newDB()
	job, _ := db.Jobs.Create(Job{KodaID: 1})

	var out Job
	if err := db.Jobs.FindByKodaID(job.KodaID).One(&out); err != nil {
		t.Fatal("failed to find job:", err)
	}

	if out.ID != job.ID {
		t.Errorf("id mismatch: %s != %s", out.ID, job.ID)
	}
}

func TestJob_AppendLog(t *testing.T) {
	db := newDB()
	job, _ := db.Jobs.Create(Job{KodaID: 1})
	db.Jobs.AppendLog(job.ID, "line goes here")

	var out Job
	db.Jobs.FindByID(job.ID).One(&out)
	if len(out.Log) != 1 {
		t.Error("log line was not appended")
	}
}
