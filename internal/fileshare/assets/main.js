const items = document.querySelectorAll(".dir-list tbody tr");
const filesInput = document.getElementById("files-input");
const createDirDialog = document.getElementById("create-dir-dialog");

let selectionIndex = 0;
const selectedClass = "selected"

let abortCtrl = null;

function isDialogOpen() {
	return Array.from(document.querySelectorAll("dialog"))
		.some(d => d.open);
}

function closeDialog() {
	return document.querySelectorAll("dialog")
		.forEach(d => d.close());
}

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

function submitWithMarked(formId) {
	const marked = getMarked();
	if (marked.length < 1) return;
	
	const form = document.getElementById(formId);
	form.innerHTML = "";
	
	const input = document.getElementById("path-input")
		  .content
		  .querySelector("input");
	
	marked.forEach(path => {
		console.log(input);
		const i = document.importNode(input, true);
		i.value = path;
		form.appendChild(i);
	});

	form.submit();
}

function download() {
	submitWithMarked("download-form");
}

function resetProgressCard() {
	document.getElementById("upload-dialog")
		.querySelector("ul").innerHTML = null;
}

function initProgressCard(name) {
	const dialog = document.getElementById("upload-dialog");

	const template = document.getElementById("upload-card")
		  .content
		  .querySelector("li");
	const card = document.importNode(template, true);
	console.log(card);
	card.querySelector(".ul-name").textContent = name;
	
	const list = dialog.querySelector("ul");
	list.appendChild(card);
	dialog.show();
	
	return card;
}

function toMB(bytes) {
	return (bytes / (1024*1024)).toFixed(1);
}

function updateProgressCard(card, transmitted, total) {
	const progress = card.querySelector("progress");
	progress.value = transmitted;
	progress.max = total;

	const count = card.querySelector(".ul-count");
	count.textContent = toMB(transmitted) + "/"
		+ toMB(total) + " MB";
}

function completeProgressCard(card, total) {
	const progress = card.querySelector("progress");
	progress.value = total;
	progress.max = total;

	const count = card.querySelector(".ul-count");
	count.textContent = "Completed";
}

function store() {
	filesInput.click();
}

const chunkSize = 1024 * 1024 // 1 MB;

async function initFileStore(abortCtrl, file) {
	console.log(file);
	const path = location.pathname
		  .split("/")
		  .slice(2)
		  .join("/");
	
	const data = new FormData();
	data.append("path", path);
	data.append("name", file.name);
	data.append("size", file.size);
	
	const response = await fetch("/file/upload", {
		method: "POST",
		body: data,
		signal: abortCtrl.signal
	});

	// Result will either be a reference to the created file or an
	// error message.
	const result = await response.text();
	if (!response.ok) {
		throw new Error(result);
    }

	return result;
}

async function storeChunk(abortCtrl, reference, file, start) {
	const end = start + chunkSize;
	
	const data = new FormData();
	data.append("chunk", file.slice(start, end));
	
	const response = await fetch("/file/upload/" + reference, {
		method: "PUT",
		body: data,
		signal: abortCtrl.signal
	});

	if (!response.ok) {
		const result = await response.text();
		throw new Error(result);
    }

	return end;
}

async function commitStoredFile(abortCtrl, reference) {
	const response = await fetch("/file/upload/" + reference, {
		method: "POST",
		signal: abortCtrl.signal
	});

	if (!response.ok) {
		const result = await response.text();
		throw new Error(result);
    }
}

async function storeFile(file) {
	const abortCtrl = getAbortController();
	const name = file.name;
	const total = file.size;
	let transmitted = 0;

	const reference = await initFileStore(abortCtrl, file);
	const ref = initProgressCard(name);

	for (transmitted = 0; transmitted < total; transmitted += chunkSize) {
		await storeChunk(abortCtrl, reference, file, transmitted);
		console.log(`uploading ${name}: ${transmitted} / ${total}`);
		updateProgressCard(ref, transmitted, total);
	}
	await commitStoredFile(abortCtrl, reference);
	completeProgressCard(ref, total);
}

async function storeFiles(files) {
	resetProgressCard();

	const promises = Array.from(files).map(storeFile);
	await Promise.all(promises);
	location.reload();
}

async function deleteMarked() {
	if (confirm("Delete all marked files?")) {
		submitWithMarked("delete-form");
	}
}

function createDirectory() {
	createDirDialog.show();
	const input = createDirDialog.querySelector("input");
	input.setSelectionRange(input.value.length, input.value.length);
}

window.addEventListener("error", function(e) {
	console.log(e);
	alert(`received error: ${e.error}`);
});

window.addEventListener("unhandledrejection", function(e) {
	if (e.reason != "cancelled") {
		console.log(e);
		alert(`received rejection: ${e.reason}`);
	}
});

filesInput.addEventListener("change", function(e) {
	if (e.target.files.length < 1) return;
	
	console.log(e.target.files);
	storeFiles(e.target.files);
	
	// TODO:
	// - display progress
	// - upload files in parallel
	// - gzip
	// - 
});

document.addEventListener("keydown", function(e) {
	let key = e.key;
	if (e.shiftKey) {
		key = key.toUpperCase();
	}
	console.log(key);
	
	if (isDialogOpen()) {
		switch (key) {
		case "Escape":
			closeDialog();
			break;
		}
	} else {
		switch (key) {
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
		case "Enter":
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
		case "s":
			store();
			break;
		case "D":
			deleteMarked();
			break;
		case "+":
			createDirectory();
			e.preventDefault();
			break;
		}
	}
});

function getAbortController() {
	if (!abortCtrl || abortCtrl.signal.aborted) {
		abortCtrl = new AbortController();
	}

	return abortCtrl;
}

function abort() {
	if (abortCtrl) {
		abortCtrl.abort("cancelled");
	}
}

function addListener(query, event, func) {
	document.querySelectorAll(query)
		.forEach(e => e.addEventListener(event, func));
}

document.addEventListener("DOMContentLoaded", function() {
	selectItem();

	addListener(".btn-cancel", "click", abort)
});
