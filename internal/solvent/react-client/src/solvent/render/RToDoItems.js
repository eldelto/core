import React from 'react'
import { DragDropContext, Droppable } from 'react-beautiful-dnd'

import RToDoItem from './RToDoItem'

export default function RToDoItems(props) {
  return (
    <DragDropContext onDragEnd={props.onDragEnd}>
      <Droppable droppableId="ToDoItemsDroppable">
        {provided => (
          <div className="ToDoItems" {...provided.droppableProps} ref={provided.innerRef}>
            {props.items.sort((a, b) => a.orderValue.value - b.orderValue.value)
              .map((item, index) => (
                <RToDoItem
                  key={item.id}
                  item={item}
                  index={index}
                  onCheck={props.onCheck}
                  onRemove={props.onRemove}
                  onRename={props.onRename}
                />
              ))}

            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </DragDropContext>
  );
}