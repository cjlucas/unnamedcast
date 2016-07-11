import React from "react";
import ReactDOM from "react-dom";
import { Router, Link } from "react-router";
import {createStore, applyMiddleware} from "redux";
import thunk from "redux-thunk";

import classNames from "classnames";
import _ from "lodash";

import reducers from "./reducers";
import JobsList from "./components/JobsList.jsx";

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
    // curRoute is the path property of the current route
    const curRoute = _.get(this.props, "children.props.route.path");
    const links = [
      {to: "jobs", text: "Jobs"},
    ].map(link => {
      const cls = classNames({
        item: true,
        active: curRoute == link.to,
      });
      return <Link key={link.to} to={link.to} className={cls}>{link.text}</Link>;
    });

    return (
      <div>
        <div className="ui menu">
          <div className="header item">
            unnamedcast
          </div>
          {links}
        </div>
        {this.props.children}

      </div>
    );
  }
}

App.propTypes = {
  children: React.PropTypes.object,
};

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
  ReactDOM.render(
    <Router routes={routes} />, document.getElementById("content"));
}

render();
store.subscribe(render);
