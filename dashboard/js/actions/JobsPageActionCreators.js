export function requestJobs() {
  return (dispatch, getState) => {
    var state = getState().selectedStateFilter;
    fetch(`/api/jobs?limit=20&state=${state || ""}&sort_by=modification_time&sort_order=desc`)
      .then(resp => resp.json())
      .then(data => dispatch({
        type: "RECEIVED_JOBS",
        jobs: data,
      }));
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

export function displayJobEntry(job) {
  return {
    type: "DISPLAY_JOB_MODAL",
    job: job,
  };
}

export function modalDismissed() {
  return { type: "MODAL_DISMISSED" };
}

