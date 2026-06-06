document.querySelectorAll('.js-toggle-details').forEach(function(btn) {
  btn.addEventListener('click', function() {
    var raw = btn.getAttribute('data-details');
    var formatted;
    try {
      formatted = JSON.stringify(JSON.parse(raw), null, 2);
    } catch(e) {
      formatted = raw;
    }
    var overlay = document.createElement('div');
    overlay.className = 'job-details-overlay';
    overlay.innerHTML = '<div class="job-details-panel">' +
      '<button class="close-btn">&times;</button>' +
      '<pre>' + formatted.replace(/</g, '&lt;').replace(/>/g, '&gt;') + '</pre>' +
      '</div>';
    document.body.appendChild(overlay);
    overlay.querySelector('.close-btn').addEventListener('click', function() {
      overlay.remove();
    });
    overlay.addEventListener('click', function(e) {
      if (e.target === overlay) overlay.remove();
    });
  });
});
