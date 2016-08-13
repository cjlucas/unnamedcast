import {combineReducers} from "redux";

function selectedStateFilter(state = null, action) {
  if (action.type == "SELECTED_FILTER") {
    // If same state button was preseed, toggle it
    return state != action.filter
    ? action.filter
    : null;
  }

  return state;
}

function jobs(state = [], action) {
  if (action.type == "RECEIVED_JOBS") {
    var jobs = action.jobs || [];
    jobs.forEach(job => {
      job.modification_time = new Date(job.modification_time);

      job.log = job.log || [];
      job.log.forEach(log => {
        log.time = new Date(log.time);
      });
    });

    return jobs;
  }

  return state;
}

function queueStats(state = [], action) {
  if (action.type == "RECEIVED_QUEUE_STATS") {
    return action.stats || [];
  }
  return state;
}

function displayedJob(state = null, action) {
  switch (action.type) {
  case "DISPLAY_JOB_MODAL":
    return action.job;
  case "MODAL_DISMISSED":
    return null;
  default:
    return state;
  }
}

export default combineReducers({
  selectedStateFilter,
  jobs,
  queueStats,
  displayedJob,
});
