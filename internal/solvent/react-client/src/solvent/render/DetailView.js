import React from 'react';
import { Link } from "react-router-dom";

import RToDoList from './RToDoList'

import { ReactComponent as Menu } from '../../icons/menu.svg'
import { ReactComponent as BackArrow } from '../../icons/arrow-left.svg'

export default function DetailView(props) {
  return (
    <div className="DetailView">
      <Header />
      <div className="DetailViewMain">
        {props.toDoList ?
          <RToDoList
            toDoList={props.toDoList}
            checkItem={props.checkItem}
            addItem={props.addItem}
            removeItem={props.removeItem}
            moveItem={props.moveItem}
            renameItem={props.renameItem}
            renameList={props.renameList}
          />
          : ""}
      </div>
    </div>
  );
}

function Header(props) {
  return (
    <header>
      <div className="DetailViewHeader header">
        <Link to="/">
          <button className="DetailViewMenuButton menuButton">
            <BackArrow />
          </button>
        </Link>

        <h1 className="HeaderTitle">Solvent</h1>

        <button className="DetailViewMenuButton menuButton" onClick={props.onMenuClick}>
          <Menu />
        </button>
      </div>
    </header>
  );
}
