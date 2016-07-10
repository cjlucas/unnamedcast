import React from "react";
import ReactDOM from "react-dom";
import {createStore, applyMiddleware} from "redux";
import thunk from "redux-thunk";

import reducers from "./reducers";
import JobsList from "./components/JobsList.jsx";

console.log('here', reducers);

const store = createStore(reducers, applyMiddleware(thunk));

function render() {
  ReactDOM.render(
    <JobsList store={store} />, document.getElementById("content"));
}

render();
store.subscribe(render);
