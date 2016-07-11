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
          <Link to="/jobs" className="item">Jobs</Link>
          <a className="item active">
            Locations
          </a>
        </div>
        {this.props.children}

      </div>
    );
  }
}


const routes = {
  path: "/",
  component: App,
  childRoutes: [
    {
      path: "jobs",
      component: JobsListWrapper,
    }
  ]
};

function render() {
  // ReactDOM.render(
  //   <JobsList store={store} />, document.getElementById("content"));
  ReactDOM.render(
    <Router routes={routes} />, document.getElementById("content"));
}


render();
store.subscribe(render);
