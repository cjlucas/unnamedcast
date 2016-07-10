import React from "react";
import classNames from "classnames";

import AppDispatcher from "../dispatcher/AppDispatcher";
import JobsListActionCreators from "../actions/JobsListActionCreators";

class Button extends React.Component {
  _onClick(key) {
    console.log(`did click ${key}`);
    JobsListActionCreators.selectedFilter(key);
  }

  render() {
    var cls = {
      ui: true,
      button: true,
      basic: !this.props.selected,
    };
    cls[this.props.color] = true;

    cls = classNames(cls);

    return (
      <button className={cls} onClick={this._onClick}>
        {this.props.text}
      </button>
    );
  }
}

class QueueFilterButtons extends React.Component {
  constructor() {
    super(...arguments);
    this.state = {
      buttonStates: {
        queued: false,
        working: false,
        finished: false,
        dead: false,
      }
    };
  }

  componentWillUpdate() {
    for (var key in this.state.buttonStates) {
      this.state.buttonStates[key] = false;
    }
  }

  render() {
    var buttons = [
      {key: "queued", text: "Queued", color: "teal"},
      {key: "working", text: "Working", color: "purple"},
      {key: "finished", text: "Finished", color: "green"},
      {key: "dead", text: "Dead", color: "red"},
    ].map(info => {
      return (
        <Button
          name={info.key}
          key={info.key}
          selected={this.state.buttonStates[info.key]}
          text={info.text}
          color={info.color} />
      );
    });

    return (
      <div>
        {buttons}
      </div>
    );
  }
}

QueueFilterButtons.dispatchToken = AppDispatcher.dispatch(action => {
  console.log("Got action");
  console.log(action);
});

class JobEntry extends React.Component {
  render() {
    var title;
    var icon;
    switch(this.props.state) {
    case "finished":
      title = "Finished";
      icon = "checkmark";
      break;
    case "queued":
      title = "Queued";
      icon = "hourglass half";
      break;
    case "dead":
      title = "Dead";
      icon = "remove";
      break;
    case "working":
      title = "Working";
      icon = "refresh";
      break;
    default:
      title = `Unknown: ${this.props.state}`;
      icon = "help";
    }

    icon += " icon";

    return (
      <tr>
        <td style={{textAlign: "center"}} className="collapsing">
          <i title={title} className={icon}></i>
        </td>
        <td className="collapsing">{this.props.id}</td>
        <td className="collapsing">{this.props.queue}</td>
        <td className="mono">{JSON.stringify(this.props.payload)}</td>
        <td className="collapsing">{this.props.completionTime}</td>
      </tr>
    );
  }
}

export default class JobsList extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      stateFilters: [],
      jobs: [],
    };
  }

  render() {
    var jobs = this.state.jobs.map(job => {
      return (
        <JobEntry
          key={job.id}
          id={job.id}
          queue={job.queue}
          state={job.state}
          payload={job.payload}
          completionTime={job.completion_time}/>
      );
    });

    return (
      <div>
        <QueueFilterButtons />

        <table className="ui celled table">
          <thead>
            <tr>
              <th>State</th>
              <th>Job ID</th>
              <th>Queue</th>
              <th>Payload</th>
              <th>Completion Time</th>
            </tr>
          </thead>
          <tbody>
            {jobs}
          </tbody>
        </table>
      </div>
    );
  }
}
