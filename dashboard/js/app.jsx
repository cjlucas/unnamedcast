import React from "react";
import ReactDOM from "react-dom";
import { Router, Link } from "react-router";
import {createStore, applyMiddleware} from "redux";
import thunk from "redux-thunk";

import classNames from "classnames";
import _ from "lodash";

import reducers from "./reducers";
import JobsPage from "./containers/JobsPage.jsx";

const store = createStore(reducers, applyMiddleware(thunk));

class JobsPageWrapper extends React.Component {
  render() {
    return (
      <JobsPage store={store} />
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
      component: JobsPageWrapper,
    }
  ]
};

function render() {
  ReactDOM.render(
    <Router routes={routes} />, document.getElementById("content"));
}

render();
store.subscribe(render);
