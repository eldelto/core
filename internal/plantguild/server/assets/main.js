// TODO: Remove the whole good-companions list and display all suggestions
// sorted by decreasing rating. Rating = 100 * plantNameLen / queryLen + goodCompanionConstant
// This properly handles search ranking and empty search fields

let selectedPlants = new Set();
let suggestions = new Array();
let goodCompanions = new Set();
let badCompanions = new Set();

function ratePlant(plantName, query) {
  return 100 * (query.length / plantName.length) + goodCompanions.has(plantName) * 10;
}

function populateSuggestions(query) {
  query = query.toLowerCase();
  const suggestionRanking = new Array();
  plantGuilds.forEach((_, key) => {
    if (key.toLowerCase().indexOf(query) >= 0) {
      suggestionRanking.push({ name: key, rating: ratePlant(key, query) });
    }
  })

  suggestions = suggestionRanking.sort((a, b) => b.rating - a.rating)
    .map((s) => s.name);
}

function updateSuggestions(query) {
  const list = window.document.getElementById("suggestions");
  list.innerHTML = "";
  suggestions.forEach((plantName) => {
    const li = document.createElement("li");
    li.onclick = () => selectPlant(plantName);
    li.textContent = plantName;
    if (goodCompanions.has(plantName)) li.classList.add("good-companion");
    if (badCompanions.has(plantName)) li.classList.add("bad-companion");
    list.appendChild(li);
  });
}

function search() {
  const searchInput = window.document.getElementById("plant-search");
  populateSuggestions(searchInput.value);
  updateSuggestions(searchInput.value);
}

function updateSelectedPlants(query) {
  const list = window.document.getElementById("selected-plants"); // TODO: Move selection to on-page-load
  list.innerHTML = "";
  selectedPlants.forEach((plantName) => {
    const li = document.createElement("li");
    list.appendChild(li);

    const span = document.createElement("span");
    span.textContent = plantName;
    li.appendChild(span);

    const button = document.createElement("button");
    button.onclick = () => removePlant(plantName);
    button.textContent = 'x';
    li.appendChild(button);
  });

  populateCompanions();
}

function selectFirstPlant() {
  if (suggestions.length > 0) {
    const plantName = suggestions[0];
    selectPlant(plantName);
  }

  return false;
}

function selectPlant(name) {
  selectedPlants.add(name);
  updateSelectedPlants();
  const searchInput = window.document.getElementById("plant-search");
  searchInput.value = "";
  search();
}

function removePlant(name) {
  selectedPlants.delete(name);
  updateSelectedPlants();
}

function populateCompanions() {
  goodCompanions = new Set();
  badCompanions = new Set();

  selectedPlants.forEach((plantName) => {
    const plantInfo = plantGuilds.get(plantName);

    // Union
    goodCompanions = new Set([...goodCompanions, ...plantInfo.GoodCompanions]);
    badCompanions = new Set([...badCompanions, ...plantInfo.BadCompanions]);
  });

  for (var x of badCompanions) if (goodCompanions.has(x)) goodCompanions.delete(x);
  for (var x of selectedPlants) {
    if (goodCompanions.has(x)) goodCompanions.delete(x);
    if (badCompanions.has(x)) badCompanions.delete(x);
  }
}
