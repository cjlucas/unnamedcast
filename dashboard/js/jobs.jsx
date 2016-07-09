import React from 'react';
import _ from 'lodash';
import classNames from 'classnames';

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
    default:
      title = `Unknown: ${this.props.state}`;
      icon = "help";
    }

    icon += " icon";

    return (
      <tr>
        <td style={{textAlign: 'center'}} className="collapsing">
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

class Button extends React.Component {
  render() {
    var cls = {
      ui: true,
      button: true,
      basic: !this.props.selected,
    }
    cls[this.props.color] = true;

    cls = classNames(cls);

    return (
      <button className={cls} onClick={() => this.props.onClick(this.props.name)}>
        {this.props.text}
      </button>
    );
  }
}

class QueueFilterButtons extends React.Component {

  constructor() {
    super();
    this.state = {
      buttonStates: {
        queued: false,
        working: false,
        finished: false,
        dead: false,
      }
    };
  }

  onButtonClick(key) {
    var states = {
      queued: false,
      working: false,
      finished: false,
      dead: false,
    };
    states[key] = true;
    this.setState({buttonStates: states});
    this.props.onChange(key);
  }

  render() {
    var buttons = [
      {key: "queued", text: "Queued", color: "brown"},
      {key: "working", text: "Working", color: "purple"},
      {key: "finished", text: "Finished", color: "green"},
      {key: "dead", text: "Dead", color: "red"},
    ].map(info => {
      return (
        <Button
          onClick={this.onButtonClick.bind(this)}
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

export class JobList extends React.Component {
  constructor() {
    super(...arguments);
    this.state = {
      jobs: [],
      stateFilters: [],
    };
  }

  componentWillMount() {
    this.fetchJobs();
    setInterval(this.fetchJobs.bind(this), 5000)
  }

  onStateFilterChanged(selectedStates) {
    this.state.stateFilters = [selectedStates];
    this.fetchJobs();
    this.setState(this.state);
  }

  fetchJobs() {
    var params = {
      limit: 20,
    }

    if (this.state.stateFilters.length > 0) {
      params.state = this.state.stateFilters[0];
    }

    var s = '';
    Object.keys(params).forEach(key => s += `${key}=${params[key]}&`)

    console.log(s);

    fetch(`/api/jobs?${s}`)
    .then(resp => resp.json())
    .then(data => this.setState({jobs: data || []}));
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
        <QueueFilterButtons onChange={this.onStateFilterChanged.bind(this)}/>

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
