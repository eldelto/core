document.addEventListener("DOMContentLoaded", function () {
	console.log("<3");
	let copybutton = document.querySelector("#copy-button");
	console.log(copybutton);
});

function copyAll() {
	let table = document.querySelector("#expenses");
	let text = "";
	let rows = table.querySelectorAll("tr")
	for (let row of rows) {
		let data = row.querySelectorAll("td")
		for (let inhalt of data) {
			text = text + "\t" + inhalt.textContent
		}
		text = text + "\n"
	}

	navigator.clipboard.writeText(text);
	window.alert("Erfolgreich!");

}
