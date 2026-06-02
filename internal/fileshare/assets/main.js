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

function getLink(item) {
	return item.querySelector("a");
}

function getPath(item) {
	return item.querySelector("input[name=path]");
}

function navigateUp() {
	const parent = document.getElementById("parent");
	if (parent) {
		parent.click();
	}
}

function navigateDown() {
	const link = getLink(items[selectionIndex]);
	if (link) {
		link.click();
	}
}

function isMarked(item) {
	const cb = item.querySelector("input[type=checkbox]");
	return cb && cb.checked;
}

function getMarked() {
	const paths = [];
	items.forEach(i => {
		if (isMarked(i)) {
			paths.push(getPath(i).value);
		}
	});

	return paths;
}

function download() {
	const form = document.getElementById("download-form");
	form.innerHTML = "";
	
	const input = document.getElementById("path-input")
		  .content
		  .querySelector("input");
	
	getMarked().forEach(path => {
		console.log(input);
		const i = document.importNode(input, true);
		i.value = path;
		form.appendChild(i);
	});

	form.submit();
}

document.addEventListener("keydown", function(e) {
	let key = e.key;
	if (e.shiftKey) {
		key = key.toUpperCase();
	}
	console.log(key);
	
	switch (key) {
	case "Enter":
	case "ArrowDown":
	case "j":
		selectionIndex = moveIndex(1);
		selectItem();
		break;
	case "ArrowUp":
	case "k":
		selectionIndex = moveIndex(-1);
		selectItem();
		break;
	case "ArrowLeft":
	case "h":
		navigateUp();
		break;
	case "ArrowRight":
	case "l":
		navigateDown();
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
	case "d":
		download();
		break;
	}
});

document.addEventListener("DOMContentLoaded", function() {
	selectItem();
});
