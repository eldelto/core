document.addEventListener("DOMContentLoaded", function () {
    let grid = document.querySelector(".grid-5x6");
    grid.addEventListener("click", function (event) {
    let tile = event.target.closest(".tile");
    tile.classList.add("selected")
    })
})
