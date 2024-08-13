document.addEventListener("DOMContentLoaded", function () {
	console.log("<3");
	let copybutton = document.querySelector("#copy-button");
	console.log(copybutton);
});

function copyAll() {
	let table = document.querySelector("#expenses");
	let text = "";
	let rows = table.querySelectorAll("tr")
	for(let row of rows) {
		let data = row.querySelectorAll("td")
		for(let inhalt of data) {console.log(inhalt.textContent)}
	}
	



	
	;
	navigator.clipboard.writeText(table.textContent);
	window.alert("Erfolgreich!");


}
