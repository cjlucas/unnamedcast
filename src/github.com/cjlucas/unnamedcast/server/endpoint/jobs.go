package endpoint

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/middleware"
	"github.com/gin-gonic/gin"
)

type GetJobs struct {
	DB     *db.DB
	Query  db.Query
	Params struct {
		sortParams
		limitParams
		Queue string `param:"queue"`
		State string `param:"state"`
	}
}

func (e *GetJobs) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.AddQuerySortInfo(e.DB.Jobs.ModelInfo, &e.Query, &e.Params, "modification_time"),
		middleware.AddQueryLimitInfo(&e.Query, &e.Params),
	}
}

func (e *GetJobs) Handle(c *gin.Context) {
	e.Query.Filter = make(db.M)
	if e.Params.Queue != "" {
		e.Query.Filter["queue"] = e.Params.Queue
	}
	if e.Params.State != "" {
		e.Query.Filter["state"] = e.Params.State
	}

	var jobs []db.Job
	if err := e.DB.Jobs.Find(&e.Query).All(&jobs); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

type GetJob struct {
	DB  *db.DB
	Job db.Job
}

func (e *GetJob) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Jobs,
			BoundName:  "id",
			Result:     &e.Job,
		}),
	}
}

func (e *GetJob) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, &e.Job)
}

type CreateJob struct {
	DB   *db.DB
	Koda *koda.Client
	Job  db.Job
}

func (e *CreateJob) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.UnmarshalBody(&e.Job),
	}
}

func (e *CreateJob) Create() (db.Job, error) {
	j, err := e.Koda.CreateJob(e.Job.Payload)
	if err != nil {
		return db.Job{}, fmt.Errorf("failed to create koda job: %s", err)
	}

	e.Job.KodaID = j.ID
	e.Job.State = "initial"
	job, err := e.DB.Jobs.Create(e.Job)
	if err != nil {
		return db.Job{}, fmt.Errorf("failed to create job db entry: %s", err)
	}

	j, err = e.Koda.SubmitJob(koda.Queue{Name: job.Queue}, job.Priority, j)
	if err != nil {
		return db.Job{}, fmt.Errorf("failed to submit job: %s", err)
	}

	if err := e.DB.Jobs.UpdateState(job.ID, "queued"); err != nil {
		return db.Job{}, fmt.Errorf("failed to update state: %s", err)
	}

	return job, nil
}

func (e *CreateJob) Handle(c *gin.Context) {
	if e.Job.Queue == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("queue not specified"))
		return
	}

	job, err := e.Create()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &job)
}
