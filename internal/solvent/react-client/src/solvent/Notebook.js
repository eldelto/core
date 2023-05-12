import { v4 as uuid } from 'uuid'
import PSet from './PSet';
import ToDoList from './ToDoList';

export default class Notebook {

  constructor(id, toDoLists, createdAt) {
    this.id = id;
    this.toDoLists = toDoLists;
    this.createdAt = createdAt;
  }

  static new() {
    return new Notebook(uuid(), PSet.new("ToDoListPSet"), currentNanos());
  }

  addList(title) {
    const list = ToDoList.new(title);
    this.toDoLists.add(list);

    return list;
  }

  removeList(id) {
    const list = this.getList(id);
    if (list) {
      this.toDoLists.remove(list);
    }
  }

  getList(id) {
    return this.toDoLists.liveView().get(id);
  }

  getLists() {
    const lists = [];
    this.toDoLists.liveView().forEach((item, _) => lists.push(item));

    return lists;
  }

  identifier() {
    return this.id;
  }

  merge(other) {
    if (this.identifier() !== other.identifier()) {
      throw "Notebooks with different IDs cannot be merged";
    }

    const mergedToDoLists = this.toDoLists.merge(other.toDoLists);

    const mergedNotebook = new Notebook(this.id, mergedToDoLists, this.createdAt);
    return mergedNotebook;
  }
}

function currentNanos() {
  return Date.now() * 1000000;
}