import AppDispatcher from "../dispatcher/AppDispatcher";

export function selectedFilter(filter) {
  return AppDispatcher.dispatch({
    type: "selected_filter",
    filter: filter,
  });
}
