import React from "react";
import Portal from "react-portal";

export default class Modal extends React.Component {
  _modal(action) {
    $(this.refs.modal).modal(action); // eslint-disable-line
  }

  _show() {
    this._modal("show");
  }
  
  _hide() {
    this._modal("hide");
  }

  render() {
    const { header, content, actions } = this.props;

    var actionsElem;
    if (actions) {
      actionsElem = <div className="actions">{actions}</div>;
    }

    return (
      <Portal
        isOpened={this.props.isOpened}
        closeOnEsc
        closeOnOutsideClick
        ref="portal"
        beforeClose={(node, removeFromNode) => {
          this._hide();
          removeFromNode();
          if (this.props.onClose) this.props.onClose();
        }}
        onOpen={this._show.bind(this)}>
        <div className="ui modal" ref="modal">
          <div className="header">
            {header}
          </div>
          <div className="content">
            {content}
          </div>
          {actionsElem}
        </div>
      </Portal>
    );
  }
}

Modal.propTypes = {
  isOpened: React.PropTypes.bool.isRequired,
  onClose: React.PropTypes.func,
  header: React.PropTypes.string,
  content: React.PropTypes.element,
  actions: React.PropTypes.array,
};
