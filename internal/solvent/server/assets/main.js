const deletingClass = "deleting";

function keydownEvent() { return new Event("keydown") };
function deleteItemEvent() { return new Event("delete-item") };
function itemMovedEvent() { return new Event("item-moved") };

function autosize(){
	const el = this;
	setTimeout(() => {
		// Subtract 10px so the element doesn't keep growing.
		el.style = "height:" + (el.scrollHeight - 10) + "px;overflow-y:hidden;";
	}, 10);
}

let pressTimer;
function startLongPress(e) {
	clearTimeout(pressTimer);

	const item = e.currentTarget.closest(".ToDoItem");
	item.classList.add(deletingClass);

	pressTimer = window.setTimeout(() => {
		item.classList.remove(deletingClass);
		item.dispatchEvent(deleteItemEvent());
	}, 450);
}

function cancelLongPress(e) {
	const item = e.currentTarget.closest(".ToDoItem");
	item.classList.remove(deletingClass);
	clearTimeout(pressTimer);
}

function enableAddItemButton(e) {
	const input = document.querySelector("#AddItemBarTitle");
	const button = document.querySelector("#AddItemBarButton");
	button.disabled = input.value.length < 1;
}

function init() {
	document.querySelectorAll(".AddItemBar")
		.forEach(e => {
			e.addEventListener("keyup", enableAddItemButton);
			e.addEventListener("reset", enableAddItemButton);
		});

	document.querySelectorAll(".ToDoItemCheckbox")
		.forEach(e => {
			e.addEventListener("mousedown", startLongPress);
			e.addEventListener("mousemove", cancelLongPress);
			e.addEventListener("mouseup", cancelLongPress);
			e.addEventListener("touchstart", startLongPress, {passive: true});
			e.addEventListener("touchmove", cancelLongPress, {passive: true});
			e.addEventListener("touchend", cancelLongPress, {passive: true});
		});

	document.querySelectorAll("form")
		.forEach(e => e.addEventListener("htmx:afterRequest", e => {
			if (e.detail.successful) e.target.reset();
		}));
}

htmx.onLoad(function(content) {
	const sortables = content.querySelectorAll(".sortable");
	for (var i = 0; i < sortables.length; i++) {
		const sortable = sortables[i];
		const sortableInstance = new Sortable(sortable, {
			animation: 150,
			handle: ".sortHandle",
			dragClass: "sortDrag",
			ghostClass: "sortGhost",

			// Make the `.htmx-indicator` unsortable
			filter: ".htmx-indicator",
			onMove: function (evt) {
				return evt.related.className.indexOf('htmx-indicator') === -1;
			},

			onUpdate: function (evt) {
				const input = evt.item.querySelector("input[name='index']");
				input.value = evt.newIndex;
				evt.item.dispatchEvent(itemMovedEvent());
			},

			// Disable sorting on the `end` event
			onEnd: function (evt) {
				if (evt.newIndex !== evt.oldIndex) {
					this.option("disabled", true);
				}
			}
		});

		// Re-enable sorting on the `htmx:afterSwap` event
		sortable.addEventListener("htmx:afterRequest", function() {
			sortableInstance.option("disabled", false);
		});
	}
})

document.addEventListener("DOMContentLoaded", function() {
	// Resize textarea to fit the contained content.
	document.querySelectorAll("textarea[data-auto-grow]")
		.forEach(e => {
			e.addEventListener("keydown", autosize);
			e.dispatchEvent(keydownEvent());
		});

	// Move cursor to the end of the textarea.
	document.querySelectorAll("textarea[autofocus]")
		.forEach(e => {
			e.selectionStart = e.value.length;
			setTimeout(() => e.scrollIntoView({behavior: "smooth", block: "end"}), 10);
		});

	// TODO: Move shortcuts into HTMX.
	document.querySelectorAll("form[data-quick-submit]")
		.forEach(e => {
			e.addEventListener("keydown", e => {
				if (e.key == "Enter" && (e.ctrlKey || e.shiftKey)) {
					e.currentTarget.submit();
					e.preventDefault();
				}
			});
		});

	const body = document.querySelector("body");
	body.addEventListener("keydown", e => {
		if (e.key == "e" && e.ctrlKey) {
			const editLink = document.querySelector("#edit-link");
			editLink && editLink.click();
			e.preventDefault();
		}
	});
	body.addEventListener("keydown", e => {
		if (e.key == "a" && e.ctrlKey) {
			const addItemInput = document.querySelector("#AddItemBarTitle");
			addItemInput.focus();
			e.preventDefault();
		}
	});
	body.addEventListener("htmx:afterSwap", e => {
		init();
	});

	init();

	htmx.on("htmx:responseError", function (evt) {
		const dialog = document.querySelector("#error-dialog");
		const dialogContent = document.querySelector("#error-content");
		dialogContent.innerHTML = evt.detail.xhr.response;
		dialog.showModal();
    });
});
