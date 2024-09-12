function isTileEmpty(tile) {
	return tile.classList.contains("empty");
}

document.addEventListener("DOMContentLoaded", function () {

	let firstSelectedTile = null;
	let comboCounter = 0;
	let grid = document.querySelector(".grid-5x6");
	grid.addEventListener("click", function (event) {
		let clickedTile = event.target.closest(".tile");

		// Skip the remaining logic if the clicked tile is already
		// empty.
		if (isTileEmpty(clickedTile)) {
			return;
		}

		if (firstSelectedTile == null) {
			// Select the first tile if none is yet selected.
			firstSelectedTile = clickedTile;
			firstSelectedTile.classList.add("selected");
		} else {
			// Otherwise select the second tile.
			let secondSelectedTile = clickedTile;

			// Match the children of the two tiles and remove all
			// matches.
			let matched = false;
			let children1 = Array.from(firstSelectedTile.children);
			let children2 = Array.from(secondSelectedTile.children);
			for (const child1 of children1) {
				for (const child2 of children2) {
					if (child1.className == child2.className) {
						matched = true;
						firstSelectedTile.removeChild(child1);
						secondSelectedTile.removeChild(child2);
					}
				}
			}
			if (matched == true) {
				comboCounter = comboCounter + 1;

			} else {
				comboCounter = 0;
			}
			document.querySelector("#comboCounter").textContent = comboCounter;

			if (firstSelectedTile.childElementCount == 0) {
				firstSelectedTile.classList.add("empty");
			}
			if (secondSelectedTile.childElementCount == 0) {
				secondSelectedTile.classList.add("empty");
			}

			// TODO: Deselect the tile if there are no children left.

			// Keep second tile selected if we've got a match and it
			// is not empty.
			firstSelectedTile.classList.remove("selected");
			if (matched && !isTileEmpty(secondSelectedTile)) {
				firstSelectedTile = secondSelectedTile;
				firstSelectedTile.classList.add("selected");
			} else {
				firstSelectedTile = null;
			}
		}
	});
});
