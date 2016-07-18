function receivedJobs(jobs) {
  return {
    type: "RECEIVED_JOBS",
    jobs: jobs
  };
}

export function requestJobs() {
  return (dispatch, getState) => {
    var state = getState().selectedStateFilter;
    fetch(`/api/jobs?limit=20&state=${state || ""}`)
      .then(resp => resp.json())
      .then(data => dispatch(receivedJobs(data)));
  };
}

export function selectedFilter(filter) {
  return {
    type: "SELECTED_FILTER",
    filter: filter,
  };
}

function receivedQueueStats(stats) {
  return {
    type: "RECEIVED_QUEUE_STATS",
    stats: stats
  };
}

export function fetchQueueStats(times) {
  return (dispatch) => {
    fetch(`/api/stats/queues?ts=${times.join(",")}`)
      .then(resp => resp.json())
      .then(data => dispatch(receivedQueueStats(data)));
  };
}
