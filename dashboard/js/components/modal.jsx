import React from "react";
import ReactDOM from "react-dom";


export default class Modal extends React.Component {
  show() {
    var e = ReactDOM.findDOMNode(this.refs.modal);
    $(e).modal('show');
  }
  
  hide() {
    var e = ReactDOM.findDOMNode(this.refs.modal);
    $(e).modal('hide');
  }

  render() {
    return (
      <div className="ui modal" ref="modal">
        {this.props.children}
      </div>
    );
  }
}
