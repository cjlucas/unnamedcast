function receivedJobs(jobs) {
  return {
    type: "RECEIVED_JOBS",
    jobs: jobs
  };
}

export function requestJobs() {
  console.log("requestJobs");
  return (dispatch, getState) => {
    var state = getState().selectedStateFilter;
    console.log(state);
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
