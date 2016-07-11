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
    return action.jobs || [];
  }

  return state;
}


export default combineReducers({
  selectedStateFilter,
  jobs
});
