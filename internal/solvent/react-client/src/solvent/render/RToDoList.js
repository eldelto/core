import React from 'react'

import AddItemBar from './AddItemBar'
import RToDoItems from './RToDoItems'
import InputArea from './InputArea'

export default class RToDoList extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      editing: false,
      newTitle: this.props.toDoList.title.value,
      newItemTitle: ""
    }
  }

  startTitleEditing = () => {
    this.setState({
      editing: true,
      newTitle: this.props.toDoList.title.value
    });
  }

  editListTitle = event => {
    this.setState({ newTitle: event.target.value });
  }

  renameList = () => {
    this.props.renameList(this.props.toDoList, this.state.newTitle);
    this.setState({
      editing: false,
      newTitle: this.props.toDoList.title.value
    });
  }

  checkItem = (item) => this.props.checkItem(this.props.toDoList, item);

  addItem = event => {
    this.props.addItem(this.props.toDoList, this.state.newItemTitle);
    this.setState({ newItemTitle: "" });
    event.preventDefault();
  }

  removeItem = (item) => this.props.removeItem(this.props.toDoList, item);

  renameItem = (item, title) => this.props.renameItem(this.props.toDoList, item, title);

  editNewItemTitle = event => {
    this.setState({ newItemTitle: event.target.value });
  }

  moveItem = result => {
    const { destination, source, draggableId } = result;

    if (!destination) {
      return;
    }

    if (destination.droppableId === source.droppableId &&
      destination.index === source.index) {
      return;
    }

    this.props.moveItem(this.props.toDoList, draggableId, destination.index);
  }

  render() {
    return (
      <div className={"ToDoList" + (this.props.toDoList.isChecked() ? " checked" : "")}>
        <InputArea
          className="ToDoListTitle"
          value={this.state.editing ? this.state.newTitle : this.props.toDoList.title.value}
          placeholder="Title"
          onFocus={this.startTitleEditing}
          onChange={this.editListTitle}
          onBlur={this.renameList}
          onEnter={() => document.activeElement.blur()}
        />

        <div className="ToDoListBody">
          <RToDoItems
            items={this.props.toDoList.items}
            onCheck={this.checkItem}
            onRemove={this.removeItem}
            onDragEnd={this.moveItem}
            onRename={this.renameItem}
          />

          <AddItemBar
            value={this.state.newItemTitle}
            onChange={this.editNewItemTitle}
            onSubmit={this.addItem}
            disabled={this.state.newItemTitle.trim().length <= 0}
          />
        </div>
      </div>
    );
  }
}
