import React from "react";
import {Bar as BarChart} from "react-chartjs";
import _ from "lodash";
import { connect } from "react-redux";

import Button from "../components/Button.jsx";

import * as Actions from "../actions/JobsPageActionCreators";
import {shortDuration} from "../util/time";

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
      labels: times.map(shortDuration),
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

QueueList.propTypes = {
  stats: React.PropTypes.array,
};

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

QueueFilterButtons.propTypes = {
  selectedButton: React.PropTypes.string,
  onFilterSelected: React.PropTypes.func,
};

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
        <td className="collapsing">{this.props.modificationTime}</td>
      </tr>
    );
  }
}

JobEntry.propTypes = {
  id: React.PropTypes.string,
  queue: React.PropTypes.string,
  payload: React.PropTypes.object,
  modificationTime: React.PropTypes.string,
  state: React.PropTypes.string,
};

class JobsTable extends React.Component {
  render() {
    var jobs = this.props.jobs.map(job => {
      return (
        <JobEntry
          key={job.id}
          id={job.id}
          queue={job.queue}
          state={job.state}
          payload={job.payload}
          modificationTime={job.modification_time}/>
      );
    });

    return (
      <table className="ui celled table">
        <thead>
          <tr>
            <th>State</th>
            <th>Job ID</th>
            <th>Queue</th>
            <th>Payload</th>
            <th>Modification Time</th>
          </tr>
        </thead>
        <tbody>
          {jobs}
        </tbody>
      </table>
    );
  }
}

JobsTable.propTypes = {
  jobs: React.PropTypes.array.isRequired,
};

class JobsPage extends React.Component {
  fetchData() {
    this.props.requestJobs();
    this.props.fetchQueueStats();
  }

  componentWillMount() {
    this.fetchData();
    setInterval(this.fetchData.bind(this), 2000);
  }

  render() {
    const { selectedStateFilter, queueStats, jobs } = this.props;
    return (
      <div>
        <div className="ui container">
          <h1 className="ui header">Queues</h1>
          <QueueList stats={queueStats}/>
        </div>
        <div className="ui container">
          <h1 className="ui header">Jobs</h1>
          <QueueFilterButtons
            selectedButton={selectedStateFilter}
            onFilterSelected={filter => this.props.selectedFilter(filter)} />
          <JobsTable jobs={jobs} />
        </div>
      </div>
    );
  }
}

JobsPage.propTypes = {
  store: React.PropTypes.object.isRequired,
  selectedStateFilter: React.PropTypes.string,
  queueStats: React.PropTypes.array,
  jobs: React.PropTypes.array,
};

function mapStateToProps(state) {
  return {
    selectedStateFilter: state.selectedStateFilter,
    queueStats: state.queueStats,
    jobs: state.jobs,
  };
}

function mapDispatchToProps(dispatch) {
  return {
    selectedFilter: filter => {
      dispatch(Actions.selectedFilter(filter));
      dispatch(Actions.requestJobs());
    },
    requestJobs: () => dispatch(Actions.requestJobs()),
    fetchQueueStats: () => {
      return Actions.fetchQueueStats([
        5 * 60,
        10 * 60,
        30 * 60,
        60 * 60,
      ]);
    }
  };
}

export default connect(mapStateToProps, mapDispatchToProps)(JobsPage);
