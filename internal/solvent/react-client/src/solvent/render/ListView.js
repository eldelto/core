import React from 'react';
import { Link } from "react-router-dom";

import { ReactComponent as Magnify } from '../../icons/magnify.svg'
import { ReactComponent as Plus } from '../../icons/plus.svg'

export default function ListView(props) {
  return (
    <div className="ListView">
      <Header />
      <ListViewMain toDoLists={props.toDoLists} onAddList={props.addList} />
    </div>
  );
}

function Header(props) {
  return (
    <header>
      <div className="ListViewHeader header">
        <button className="DetailViewMenuButton menuButton" style={{ visibility: "hidden" }} onClick={props.onSearchClick}>
          <Magnify />
        </button>

        <h1 className="HeaderTitle">Solvent</h1>

        <button className="DetailViewMenuButton menuButton" onClick={props.onSearchClick}>
          <Magnify />
        </button>
      </div>
    </header>
  );
}

function ListViewMain(props) {
  return (
    <div className="ListViewMain">
      <ToDoLists
        className="ListViewToDoListsOpen"
        title="Open"
        addButton={true}
        toDoLists={props.toDoLists.filter(list => !list.isChecked())}
        onAddList={props.onAddList}
      />

      <ToDoLists
        className="ListViewToDoListsDone"
        title="Done"
        toDoLists={props.toDoLists.filter(list => list.isChecked())}
      />
    </div>
  );
}

function ToDoLists(props) {
  return (
    <div>
      <span className="ListViewToDoListsTitle">{props.title}</span>

      {props.addButton ?
        <AddListButton onClick={props.onAddList} />
        : ""}
      {props.toDoLists.sort((a, b) => b.createdAt - a.createdAt)
        .map(toDoList =>
          <ToDoList key={toDoList.id} toDoList={toDoList} />
        )}
    </div>
  );
}

function AddListButton(props) {
  return (
    <button className="ListViewAddListButton" onClick={props.onClick}>
      <Plus />
    </button>
  );
}

function ToDoList(props) {
  return (
    <Link to={"/list/" + props.toDoList.id}>
      <button className={"ListViewToDoList" + (props.toDoList.isChecked() ? " checked" : "")}>
        <span className="ListViewToDoListTitle">{props.toDoList.title.value}</span>
      </button>
    </Link>
  );
}
