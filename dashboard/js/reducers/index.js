import {combineReducers} from "redux";

const DEFAULT_STATE = {
  selectedStateFilter: null,
  jobs: [],
};

export default function filterJobState(state = DEFAULT_STATE, action) {
  switch(action.type) {
  case "SELECTED_FILTER":
    return Object.assign({}, state, {
      selectedStateFilter: action.filter,
    });

  case "RECEIVED_JOBS":
    return Object.assign({}, state, {
      jobs: action.jobs,
    });

  default:
    return state;
  }
}

// export default combineReducers(filterJobState);
