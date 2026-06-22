document.addEventListener("DOMContentLoaded", function () {
  var live = document.querySelector(".status-live");
  if (live) {
    var row = live.closest("tr");
    if (row) {
      row.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }
});
