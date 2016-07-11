import React from "react";
import ReactDOM from "react-dom";
import { Router, Route, Link} from "react-router";
import {createStore, applyMiddleware} from "redux";
import thunk from "redux-thunk";

import reducers from "./reducers";
import JobsList from "./components/JobsList.jsx";

console.log('here', reducers);

const store = createStore(reducers, applyMiddleware(thunk));

class JobsListWrapper extends React.Component {
  render() {
    return (
      <JobsList store={store} />
    );
  }
}

class App extends React.Component {
  render() {
    return (
      <div>
        <div className="ui menu">
          <div className="header item">
            Our Company
          </div>
          <a className="item">
            About Us
          </a>
          <a className="item">
            Jobs
          </a>
          <a className="item active">
            Locations
          </a>
        </div>

        <Router>
          <Route path="/" component={JobsListWrapper} />
        </Router>
      </div>
    );
  }
}

function render() {
  // ReactDOM.render(
  //   <JobsList store={store} />, document.getElementById("content"));
  ReactDOM.render(
    <App />, document.getElementById("content"));
}

render();
store.subscribe(render);
