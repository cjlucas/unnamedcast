import React from "react";
import classNames from "classnames";
import {Bar as BarChart} from "react-chartjs";
import _ from "lodash";

import * as Actions from "../actions/JobsListActionCreators";

class QueueList extends React.Component {
  chartForQueue(queue) {
    var states = [
      {name: "queued", color: [0, 181, 173]},
      {name: "working", color: [163, 51, 200]},
      {name: "finished", color: [33, 186, 69]},
      {name: "dead", color: [219, 40, 40]},
    ];

    var times = Object.keys(queue.jobs);

    var datasets = states.map(state => {
      var fillColor = state.color.concat([0.2]).join(",");
      var strokeColor = state.color.concat([0.2]).join(",");

      return {
        label: _.upperFirst(state.name),
        data: times.map(time => queue.jobs[time][state.name]),
        fillColor: `rgba(${fillColor})`,
        strokeColor: `rgba(${strokeColor})`,
      };
    });

    var data = {
      labels: times,
      datasets: datasets,
    };

    return (
      <div key={queue.name} className="eight wide column">
        <h3>{queue.name}</h3>
        <BarChart data={data} width="500%" height="300%" />
      </div>
    );
  }

  render() {
    // Label will be each time series
    // Each state will be its own dataset
    const {stats} = this.props;

    var charts = _.sortBy(stats, "name").map(this.chartForQueue);

    return (
      <div className="ui grid container center aligned">
        {charts}
      </div>
    );
  }
}

class Button extends React.Component {
  render() {
    var cls = {
      ui: true,
      button: true,
      basic: !this.props.selected,
    };
    cls[this.props.color] = true;

    cls = classNames(cls);

    return (
      <button className={cls} onClick={this.props.onClick}>
        {this.props.text}
      </button>
    );
  }
}

class QueueFilterButtons extends React.Component {
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
          selected={this.props.selectedButton == info.key}
          onClick={() => this.props.onFilterSelected(info.key)}
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
      stateFilter: this.getSelectedFilter()
    };
  }

  getState() {
    return this.props.store.getState();
  }

  getSelectedFilter() {
    return this.getState().selectedStateFilter;
  }

  fetchData() {
    this.props.store.dispatch(Actions.requestJobs());
    this.props.store.dispatch(Actions.fetchQueueStats([
      5 * 60,
      10 * 60,
      30 * 60,
    ]));
  }

  componentWillMount() {
    this.fetchData();
    setInterval(this.fetchData.bind(this), 2000);
  }

  componentWillUpdate() {
    var filter = this.getSelectedFilter();
    if (filter != this.state.stateFilter) {
      this.props.store.dispatch(Actions.requestJobs());
    }
    this.state.stateFilter = filter;
  }

  render() {
    var jobs = this.getState().jobs.map(job => {
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

    const {store} = this.props;
    return (
      <div className="ui container">
        <h1>Queues</h1>
        <QueueList stats={this.getState().queueStats}/>

        <h1>Jobs</h1>
        <QueueFilterButtons
          selectedButton={this.getSelectedFilter()}
          onFilterSelected={filter => store.dispatch(Actions.selectedFilter(filter))}
          />

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
