package endpoint

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/gin-gonic/gin"
)

type GetQueueStats struct {
	DB *db.DB
}

func (e *GetQueueStats) Bind() []gin.HandlerFunc { return nil }

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
		Queue   string
		State   string
		Seconds int
		// outputs
		Count int
		Error error
	}

	now := time.Now()
	ch := make(chan fetchInfo)
	fetch := func(info fetchInfo) {
		t := now.Add(time.Duration(-info.Seconds) * time.Second)

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
		for _, seconds := range []int{3600, 3600 * 24 * 7, 3600 * 24 * 7 * 30} {
			for _, state := range states {
				go fetch(fetchInfo{
					Queue:   queue,
					State:   state,
					Seconds: seconds,
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

		key := strconv.Itoa(info.Seconds)
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
