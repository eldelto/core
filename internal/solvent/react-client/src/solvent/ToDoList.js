import ToDoItem from './ToDoItem'
import { OrderValue } from './ToDoItem'
import { v4 as uuid } from 'uuid'
import PSet from './PSet'

export default class ToDoList {

  constructor(id, title, toDoItems, createdAt) {
    this.id = id;
    this.title = title;
    this.toDoItems = toDoItems;
    this.createdAt = createdAt;
  }

  static new(title) {
    const titleClass = new Title(title, currentNanos());
    return new ToDoList(uuid(), titleClass, PSet.new("ToDoItemPSet"), currentNanos());
  }

  get items() {
    const items = [];
    this.toDoItems.liveView().forEach((item, _) => items.push(item));

    return items;
  }

  addItem(title) {
    const id = uuid();
    const orderValue = new OrderValue(this.nextOrderValue(), currentNanos());
    const item = new ToDoItem(id, title, false, orderValue);
    this.toDoItems.add(item);

    return id;
  }

  getItem(id) {
    return this.toDoItems.liveView().get(id);
  }

  removeItem(id) {
    const item = this.getItem(id);
    if (item) {
      this.toDoItems.remove(item);
    }
  }

  checkItem(id) {
    const item = this.getItem(id);
    item.checked = true;

    return id;
  }

  uncheckItem(id) {
    const item = this.getItem(id);
    this.removeItem(id);

    const newId = uuid();
    const newItem = new ToDoItem(newId, item.title, false, item.orderValue);
    this.toDoItems.add(newItem);

    return newId;
  }

  moveItem(id, targetIndex) {
    const item = this.getItem(id);
    const items = this.items.sort((a, b) => a.orderValue.value - b.orderValue.value);

    const itemOrderValue = item.orderValue.value;
    const orderValueMid = items[targetIndex].orderValue.value;
    let orderValueAdjacent;
    if (orderValueMid < itemOrderValue) {
      // Movint item up
      if ((targetIndex - 1) >= 0) {
        orderValueAdjacent = items[targetIndex - 1].orderValue.value;
      } else {
        orderValueAdjacent = 0.0;
      }
    } else if (orderValueMid > itemOrderValue) {
      // Movint item down
      if ((targetIndex + 1) < items.length) {
        orderValueAdjacent = items[targetIndex + 1].orderValue.value;
      } else {
        orderValueAdjacent = this.nextOrderValue();
      }
    } else {
      return;
    }

    const newOrderValue = new OrderValue((orderValueMid + orderValueAdjacent) / 2.0, currentNanos());
    item.orderValue = newOrderValue;
    this.toDoItems.add(item);

    return item.id;
  }

  renameItem(id, title) {
    const oldItem = this.getItem(id);
    if (oldItem.title === title) {
      return oldItem.id;
    }

    this.removeItem(oldItem.id);

    const newId = this.addItem(title);
    const newItem = this.getItem(newId);
    newItem.orderValue = oldItem.orderValue;
    newItem.updatedAt = oldItem.updatedAt;
    this.toDoItems.add(newItem);

    return newId;
  }

  rename(title) {
    const newTitle = new Title(title, currentNanos());
    this.title = newTitle;
    return this;
  }

  isChecked() {
    const items = this.items;
    if (items.length > 0 && items.find(item => !item.checked) === undefined) {
      return true;
    } else {
      return false;
    }
  }

  identifier() {
    return this.id;
  }

  merge(other) {
    let mergedTitle = this.title;
    if (other.title.updatedAt > this.title.updatedAt) {
      mergedTitle = other.title;
    }

    const mergedtoDoItems = this.toDoItems.merge(other.toDoItems);

    return new ToDoList(this.id, mergedTitle, mergedtoDoItems, this.createdAt);
  }

  nextOrderValue() {
    let orderValue = 0.0;
    this.toDoItems.liveView().forEach((item, _) => {
      if (item.orderValue.value > orderValue) {
        orderValue = item.orderValue.value;
      }
    });

    return orderValue + 10.0;
  }
}

function currentNanos() {
  return Date.now() * 1000000;
}

export class Title {

  constructor(value, updatedAt) {
    this.value = value;
    this.updatedAt = updatedAt;
  }
}