function receivedJobs(jobs) {
  return {
    type: "RECEIVED_JOBS",
    jobs: jobs
  };
}

export function requestJobs() {
  console.log("requestJobs");
  return dispatch => {
    fetch("/api/jobs?limit=20")
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
