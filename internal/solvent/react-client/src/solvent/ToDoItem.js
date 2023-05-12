export default class ToDoItem {

  constructor(id, title, checked, orderValue) {
    this.id = id;
    this.title = title;
    this.checked = checked;
    this.orderValue = orderValue;
  }

  identifier() {
    return this.id;
  }

  merge(other) {
    const checked = this.checked || other.checked

    let mergedOrderValue = this.orderValue;
    if (other.orderValue.updatedAt > this.orderValue.updatedAt) {
      mergedOrderValue = other.orderValue;
    }

    return new ToDoItem(this.id, this.title, checked, mergedOrderValue);
  }
}

export class OrderValue {

  constructor(value, updatedAt) {
    this.value = value;
    this.updatedAt = updatedAt;
  }
}