document.addEventListener("DOMContentLoaded", function () {
	console.log("<3");
	let copybutton = document.querySelector("#copy-button");
	console.log(copybutton);
});

function copyAll() {
	let table = document.querySelector("#expenses");
	navigator.clipboard.writeText(table.textContent);
	window.alert("Erfolgreich!");


}
