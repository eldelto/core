import React from 'react'
import { Draggable } from 'react-beautiful-dnd'

import { ReactComponent as CheckedCircle } from '../../icons/check-circle-outline.svg'
import { ReactComponent as CheckedCircleBlank } from '../../icons/checkbox-blank-circle-outline.svg'
import { ReactComponent as DotsVertical } from '../../icons/dots-vertical.svg'
import { ReactComponent as TrashCan } from '../../icons/delete-outline.svg'

import InputArea from './InputArea'

// Move to own component and keep newTitle in state.
// Apply renaming on focus loss
export default class RToDoItem extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      editing: false,
      newTitle: ""
    }
  }

  startEditing = () => {
    this.setState({
      editing: true,
      newTitle: this.props.item.title
    });
  }

  editItem = event => {
    this.setState({ newTitle: event.target.value });
  }

  renameItem = () => {
    this.props.onRename(this.props.item, this.state.newTitle);
    this.setState({
      editing: false,
      newTitle: ""
    });
  }

  render() {
    return (
      <Draggable draggableId={this.props.item.id} index={this.props.index}>
        {provided => (
          <div
            className={"ToDoItem" + (this.props.item.checked ? " checked" : "")}
            ref={provided.innerRef}
            {...provided.draggableProps}
          >
            <button className="ToDoItemCheckbox" onClick={() => this.props.onCheck(this.props.item)}>
              {this.props.item.checked ? <CheckedCircle /> : <CheckedCircleBlank />}
            </ button>
            <InputArea
              className="ToDoItemTitle"
              value={this.state.editing ? this.state.newTitle : this.props.item.title}
              onFocus={this.startEditing}
              onChange={this.editItem}
              onBlur={this.renameItem}
              onEnter={() => document.activeElement.blur()}
            />
            <button
              className="ToDoItemDelete"
              onClick={() => this.props.onRemove(this.props.item)}>
              <TrashCan />
            </button>
            <span className="ToDoItemDragHandle" {...provided.dragHandleProps}>
              <DotsVertical />
            </span>

          </div>
        )}
      </Draggable>
    );
  }
}