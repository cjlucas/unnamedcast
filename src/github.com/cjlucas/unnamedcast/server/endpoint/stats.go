package endpoint

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/gin-gonic/gin"
)

type GetQueueStats struct {
	DB         *db.DB
	Times      string `param:"ts,require"`
	TimeSeries []time.Duration
}

func (e *GetQueueStats) parseTimesParam(c *gin.Context) {
	split := strings.Split(e.Times, ",")
	e.TimeSeries = make([]time.Duration, len(split))

	for i, s := range split {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, errors.New("invalid times param"))
			return
		}
		e.TimeSeries[i] = time.Duration(n) * time.Second
	}
}

func (e *GetQueueStats) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		e.parseTimesParam,
	}
}

func (e *GetQueueStats) Handle(c *gin.Context) {
	states := []string{
		"initial",
		"queued",
		"working",
		"finished",
		"dead",
	}

	var queues []string
	if err := e.DB.Jobs.Find(nil).Distinct("queue", &queues); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	type fetchInfo struct {
		// inputs
		Queue    string
		State    string
		Duration time.Duration
		// outputs
		Count int
		Error error
	}

	now := time.Now()
	ch := make(chan fetchInfo)
	fetch := func(info fetchInfo) {
		t := now.Add(-info.Duration)

		query := db.Query{
			Filter: db.M{
				"queue":             info.Queue,
				"state":             info.State,
				"modification_time": db.M{"$gte": t},
			},
		}

		info.Count, info.Error = e.DB.Jobs.Find(&query).Count()
		ch <- info
	}

	numJobs := 0
	for _, queue := range queues {
		for _, dur := range e.TimeSeries {
			for _, state := range states {
				go fetch(fetchInfo{
					Queue:    queue,
					State:    state,
					Duration: dur,
				})
				numJobs++
			}
		}

	}

	type countMap map[string]map[string]int // map[time]map[state]count
	queueInfo := make(map[string]countMap)

	for i := 0; i < numJobs; i++ {
		info := <-ch

		if info.Error != nil {
			c.AbortWithError(http.StatusInternalServerError, info.Error)
			return
		}

		if _, ok := queueInfo[info.Queue]; !ok {
			queueInfo[info.Queue] = make(countMap)
		}

		key := strconv.Itoa(int(info.Duration / time.Second))
		if _, ok := queueInfo[info.Queue][key]; !ok {
			queueInfo[info.Queue][key] = make(map[string]int)
		}

		queueInfo[info.Queue][key][info.State] = info.Count
	}

	out := make([]gin.H, len(queueInfo))
	i := 0
	for queue, counts := range queueInfo {
		out[i] = gin.H{
			"name": queue,
			"jobs": counts,
		}
		i++
	}

	c.JSON(http.StatusOK, out)
}
