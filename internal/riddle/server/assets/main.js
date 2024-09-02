document.addEventListener("DOMContentLoaded", function () {

	let firstSelectedTile = null;
	let grid = document.querySelector(".grid-5x6");
	grid.addEventListener("click", function (event) {
		if (firstSelectedTile == null) {
			firstSelectedTile = event.target.closest(".tile");
			firstSelectedTile.classList.add("selected");
		} else {
			let secondSelectedTile = event.target.closest(".tile");

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

			// TODO: If there are no children make the tile unselectable.
			//       Deselect the tile if there are no children left.

			firstSelectedTile.classList.remove("selected");
			if (matched) {
				firstSelectedTile = secondSelectedTile;
				firstSelectedTile.classList.add("selected");
			} else {
				firstSelectedTile = null;
			}
		}
	});
});
