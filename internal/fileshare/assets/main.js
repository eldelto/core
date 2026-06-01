const items = document.querySelectorAll(".dir-list tbody tr");
let selectionIndex = 0;
const selectedClass = "selected"

function moveIndex(offset) {
	const newIndex = (selectionIndex + offset) % items.length;
	if (newIndex < 0) { return items.length-1; }
	return newIndex;
}

function selectItem() {
	items[moveIndex(1)].classList.remove(selectedClass);
	items[moveIndex(-1)].classList.remove(selectedClass);
	items[selectionIndex].classList.add(selectedClass);
}

function markItem() {
	items[selectionIndex]
		.querySelector("input[type=checkbox]")
		.checked = true;
	selectionIndex = moveIndex(1);
	selectItem();
}

function unmarkItem() {
	items[selectionIndex]
		.querySelector("input[type=checkbox]")
		.checked = false;
	selectionIndex = moveIndex(1);
	selectItem();
}

function unmarkAll() {
    document
		.querySelectorAll(".dir-list tbody tr input[type=checkbox]")
	    .forEach(cb => cb.checked = false);
}

document.addEventListener("keyup", function(e) {
	let key = e.key;
	if (e.shiftKey) {
		key = key.toUpperCase();
	}
	console.log(key);
	
	switch (key) {
	case "j":
		selectionIndex = moveIndex(1);
		selectItem();
		break;
	case "k":
		selectionIndex = moveIndex(-1);
		selectItem();
		break;
	case "m":
		markItem();
		break;
	case "u":
		unmarkItem();
		break;
	case "U":
		unmarkAll();
		break;
	}
});

document.addEventListener("DOMContentLoaded", function() {
	selectItem();
});
