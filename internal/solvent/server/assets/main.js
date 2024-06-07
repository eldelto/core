const keydownEvent = new Event("keydown");

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

	document.querySelector("body")
		.addEventListener("keydown", e => {
			if (e.key == "e" && e.ctrlKey) {
				const editLink = document.querySelector("#edit-link");
				editLink && editLink.click();
				e.preventDefault();
			}
		});
});

function autosize(){
	const el = this;
	setTimeout(() => {
		el.style.height = "auto";
		el.style = "height:" + (el.scrollHeight) + "px;overflow-y:hidden;";
	}, 10);
}
