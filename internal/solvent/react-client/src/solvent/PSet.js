export default class PSet {

  constructor(liveSet, tombstoneSet, identifier) {
    this.liveSet = liveSet;
    this.tombstoneSet = tombstoneSet;
    this.id = identifier;
  }

  static new(identifier) {
    return new PSet(new Map(), new Map(), identifier);
  }

  add(item) {
    addToItemMap(this.liveSet, item);
  }

  remove (item) {
    const key = item.identifier();

    if(!this.liveView().has(key)) {
      return;
    }

    this.tombstoneSet.set(key, item);
  }

  liveView() {
    const liveView = new Map();

    this.liveSet.forEach((item, key) => {
      if(!this.tombstoneSet.has(key)) {
        liveView.set(key, item);
      }
    });

    return liveView;
  }

  identifier() {
    return this.id;
  }

  merge (other) {
    if (this.identifier() !== other.identifier()) {
      throw "Cannot merge item with ID '" + this.id + "' and item with ID '" + other.identifier() + "'";
    }

    const mergedLiveSet = mergeItemMaps(this.liveSet, other.liveSet);
    const mergedTombstoneSet = mergeItemMaps(this.tombstoneSet, other.tombstoneSet);

    const mergedPSet = new PSet(mergedLiveSet, mergedTombstoneSet, this.identifier());
    return mergedPSet;
  }
}

function addToItemMap(itemMap, item) {
  const key = item.identifier();

  if (!itemMap.has(key)) {
    itemMap.set(key, item);
    return;
  }

  const oldItem = itemMap.get(key);
  const mergedItem = oldItem.merge(item);

  itemMap.set(key, mergedItem);
}

function mergeItemMaps(thiz, other) {
  const mergedItemMap = new Map();
  thiz.forEach((item, key) => mergedItemMap.set(key, item));

  other.forEach((item, _) => {
    addToItemMap(mergedItemMap, item);
  });

  return mergedItemMap;
}