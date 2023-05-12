import Notebook from './Notebook'
import ToDoList, { Title } from './ToDoList'
import ToDoItem, { OrderValue } from './ToDoItem'
import PSet from './PSet';

function orderValueFromDto(dto) {
  return new OrderValue(dto.value, dto.updatedAt);
}

function orderValueToDto(orderValue) {
  return {
    "value": orderValue.value,
    "updatedAt": orderValue.updatedAt,
  };
}

function toDoItemFromDto(dto) {
  return new ToDoItem(
    dto.id,
    dto.title,
    dto.checked,
    orderValueFromDto(dto.orderValue),
    dto.updatedAt
  );
}

function toDoItemToDto(item) {
  return {
    "id": item.id,
    "title": item.title,
    "checked": item.checked,
    "orderValue": orderValueToDto(item.orderValue),
    "updatedAt": item.updatedAt
  };
}

function toDoItemMapFromDto(dto) {
  const map = new Map();
  dto.map(toDoItemFromDto)
    .forEach(item => map.set(item.id, item));

  return map;
}

function toDoItemMapToDto(map) {
  const dto = [];
  map.forEach((item, _) => dto.push(toDoItemToDto(item)));

  return dto;
}

function toDoItemPSetFromDto(dto) {
  return new PSet(
    toDoItemMapFromDto(dto.liveSet),
    toDoItemMapFromDto(dto.tombstoneSet),
    "ToDoItemPSet"
  );
}

function toDoItemPSetToDto(pset) {
  return {
    "liveSet": toDoItemMapToDto(pset.liveSet),
    "tombstoneSet": toDoItemMapToDto(pset.tombstoneSet),
  };
}

function titleFromDto(dto) {
  return new Title(dto.value, dto.updatedAt);
}

function titleToDto(title) {
  return {
    "value": title.value,
    "updatedAt": title.updatedAt
  };
}

function toDoListFromDto(dto) {
  return new ToDoList(
    dto.id,
    titleFromDto(dto.title),
    toDoItemPSetFromDto(dto.toDoItems),
    dto.createdAt
  );
}

function toDoListToDto(list) {
  return {
    "id": list.id,
    "title": titleToDto(list.title),
    "toDoItems": toDoItemPSetToDto(list.toDoItems),
    "updatedAt": list.updatedAt,
    "createdAt": list.createdAt
  };
}

function toDoListMapFromDto(dto) {
  const map = new Map();
  dto.map(toDoListFromDto)
    .forEach(list => map.set(list.id, list));

  return map;
}

function toDoListMapToDto(map) {
  const dto = [];
  map.forEach((list, _) => dto.push(toDoListToDto(list)));

  return dto;
}

function toDoListPSetFromDto(dto) {
  return new PSet(
    toDoListMapFromDto(dto.liveSet),
    toDoListMapFromDto(dto.tombstoneSet),
    "ToDoListPSet"
  );
}

function toDoListPSetToDto(pset) {
  return {
    "liveSet": toDoListMapToDto(pset.liveSet),
    "tombstoneSet": toDoListMapToDto(pset.tombstoneSet),
  };
}

export function notebookFromDto(dto) {
  return new Notebook(
    dto.id,
    toDoListPSetFromDto(dto.toDoLists),
    dto.createdAt
  );
}

export function notebookToDto(notebook) {
  return {
    "id": notebook.id,
    "toDoLists": toDoListPSetToDto(notebook.toDoLists),
    "createdAt": notebook.createdAt
  };
}
