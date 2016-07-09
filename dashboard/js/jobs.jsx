import React from 'react';
import $ from 'jquery';

console.log($);

class JobEntry extends React.Component {
  doit() {
    console.log('got clicked', this.props.id);
    $('.ui.modal').modal('show');
  }

  render() {
    console.log(this.refs);
    return (
      <tr onClick={this.doit.bind(this)}>
        <td>{this.props.id}</td>
        <td>{this.props.queue}</td>
        <td>{this.props.state}</td>
        <td>{this.props.completionTime}</td>
      </tr>
    );
  }
}

export class JobList extends React.Component {
  constructor() {
    super(...arguments);
    this.state = {
      jobs: []
    };
  }

  componentWillMount() {
    this.fetchJobs();
  }

  fetchJobs() {
    fetch('/api/jobs?limit=20')
    .then(resp => resp.json())
    .then(data => this.setState({jobs: data}));
  }

  render() {
    console.log(this.state);
    var jobs = this.state.jobs.map(job => {
      return (
        <JobEntry
          key={job.id}
          id={job.id}
          queue={job.queue}
          state={job.state}
          completionTime={job.completion_time}/>
      );
    });

    return (
      <table className="ui celled table" ref={c => console.log(c)}>
        <thead>
          <tr>
            <th>Job ID</th>
            <th>Queue</th>
            <th>Status</th>
            <th>Completion Time</th>
          </tr>
        </thead>
        <tbody>
          {jobs}
        </tbody>
      </table>
    );
  }
}
