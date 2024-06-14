const deletingClass = "deleting";

function keydownEvent() { return new Event("keydown") };
function deleteItemEvent() { return new Event("delete-item") };

function autosize(){
	const el = this;
	setTimeout(() => {
		el.style.height = "auto";
		el.style = "height:" + (el.scrollHeight) + "px;overflow-y:hidden;";
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
	}, 750);
}

function cancelLongPress(e) {
	const item = e.currentTarget.closest(".ToDoItem");
	item.classList.remove(deletingClass);
	clearTimeout(pressTimer);
}

function init() {
	document.querySelectorAll("#AddItemBarTitle")
		.forEach(e => e.addEventListener("keyup", e => {
			const addItemButton = document.querySelector("#AddItemBarButton");
			addItemButton.disabled = e.target.value.length < 1;
		}));

	document.querySelectorAll(".ToDoItemCheckbox")
		.forEach(e => {
			e.addEventListener("mousedown", startLongPress);
			e.addEventListener("mousemove", cancelLongPress);
			e.addEventListener("mouseup", cancelLongPress);
			e.addEventListener("touchstart", startLongPress);
			e.addEventListener("touchmove", cancelLongPress);
			e.addEventListener("touchend", cancelLongPress);
		});

	document.querySelectorAll("form")
		.forEach(e => e.addEventListener("htmx:afterRequest", e => {
			if (e.detail.successful) e.target.reset();
		}));
}

htmx.onLoad(function(content) {
	var sortables = document.querySelectorAll(".sortable");
	for (var i = 0; i < sortables.length; i++) {
		const sortable = sortables[i];
		const sortableInstance = new Sortable(sortable, {
			animation: 150,
			handle: ".sort-handle",
			ghostClass: 'blue-background-class',

			// Make the `.htmx-indicator` unsortable
			filter: ".htmx-indicator",
			onMove: function (evt) {
				return evt.related.className.indexOf('htmx-indicator') === -1;
			},

			onUpdate: function (evt) {
				const input = evt.item.querySelector("input");
				input.value = evt.newIndex;
			},

			// Disable sorting on the `end` event
			onEnd: function (evt) {
				this.option("disabled", true);
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
	// Submit form on ctrl + enter.
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
	body.addEventListener("htmx:afterSwap", e => {
		init();
	});

	init();
});
