import React from 'react'

import { ReactComponent as Plus } from '../../icons/plus.svg'
import { ReactComponent as PlusCircle } from '../../icons/plus-circle.svg'

import InputArea from './InputArea'

export default function AddItemBar(props) {

  const inputFieldRef = React.createRef();
  let addButtonRef;

  return (
    <form className="AddItemBar" onSubmit={props.onSubmit} onClick={() => inputFieldRef.current.focus()}>
      <button ref={b => addButtonRef = b} className="AddItemBarButton" type="submit" value="" disabled={props.disabled}>
        {props.disabled ? <Plus /> : <PlusCircle />}
      </ button>
      <InputArea
        ref={inputFieldRef}
        className="AddItemBarTitle"
        value={props.value}
        placeholder="New item"
        onChange={props.onChange}
        onEnter={() => addButtonRef.click()}
      />
      <button className="AddItemBarLogo" type="button" disabled={true}>+</button>
    </form>
  );
}
