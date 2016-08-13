import React from "react";
import ReactDOM from "react-dom";
import Portal from "react-portal";

export default class Modal extends React.Component {
  activate() {
    this.refs.portal.openPortal();
  }

  _show() {
    $(this.refs.modal).modal('show');
  }
  
  _hide() {
    $(this.refs.modal).modal('hide');
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
}
