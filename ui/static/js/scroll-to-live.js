document.addEventListener("DOMContentLoaded", function () {
  // Priority: live match > first upcoming (future) match
  var target = document.querySelector(".status-live");
  if (!target) {
    target = document.querySelector(".status-future");
  }
  if (target) {
    var row = target.closest("tr");
    if (row) {
      row.classList.add("match-highlight");
      row.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }
});
