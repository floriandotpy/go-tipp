document.addEventListener("DOMContentLoaded", function() {
  var copyBtn = document.getElementById("copy-token-btn");
  if (copyBtn) {
    copyBtn.addEventListener("click", function() {
      var tokenInput = document.getElementById("api-token");
      if (tokenInput && navigator.clipboard) {
        navigator.clipboard.writeText(tokenInput.value).then(function() {
          copyBtn.textContent = "Kopiert!";
          setTimeout(function() { copyBtn.textContent = "Kopieren"; }, 2000);
        });
      }
    });
  }

  var generateForm = document.getElementById("generate-token-form");
  if (generateForm && generateForm.dataset.tokenExists === "true") {
    generateForm.addEventListener("submit", function(e) {
      if (!confirm("Der bestehende Token wird ungültig. Neuen Token generieren?")) {
        e.preventDefault();
      }
    });
  }

  var revokeForm = document.getElementById("revoke-token-form");
  if (revokeForm) {
    revokeForm.addEventListener("submit", function(e) {
      if (!confirm("Token wirklich widerrufen?")) {
        e.preventDefault();
      }
    });
  }
});
