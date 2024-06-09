const keydownEvent = new Event("keydown");
const deleteItemEvent = new Event("delete-item");

document.addEventListener("DOMContentLoaded", function() {
	// Resize textarea to fit the contained content.
	document.querySelectorAll("textarea[data-auto-grow]")
		.forEach(e => {
			e.addEventListener("keydown", autosize);
			e.dispatchEvent(keydownEvent);
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

function autosize(){
	const el = this;
	setTimeout(() => {
		el.style.height = "auto";
		el.style = "height:" + (el.scrollHeight) + "px;overflow-y:hidden;";
	}, 10);
}

let pressTimer;
function init() {
	document.querySelector("#AddItemBarTitle")
		.addEventListener("input", e => {
			const addItemButton = document.querySelector("#AddItemBarButton");
			if (e.target.value.length > 0) {
				addItemButton.disabled = false;
			} else {
				addItemButton.disabled = true;
			}
		});

	document.querySelectorAll(".ToDoItem")
		.forEach(e => {
			e.addEventListener("mouseup", e => {
				clearTimeout(pressTimer);
			});

			e.addEventListener("mousedown", e => {
				const item = e.currentTarget;
				pressTimer = window.setTimeout(() => {
					item.dispatchEvent(deleteItemEvent);
				}, 750);
			});
		});
}
