import React from "react";
import classNames from "classnames";

export default class Button extends React.Component {
  render() {
    var cls = {
      ui: true,
      button: true,
      basic: !this.props.selected,
    };
    cls[this.props.color] = true;

    return (
      <button className={classNames(cls)} onClick={this.props.onClick}>
        {this.props.text}
      </button>
    );
  }
}

Button.propTypes = {
  text: React.PropTypes.string,
  color: React.PropTypes.string,
  selected: React.PropTypes.bool,
  onClick: React.PropTypes.func,
};
