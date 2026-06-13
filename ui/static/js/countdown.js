(function() {
  var el = document.getElementById('countdown');
  if (!el) return;

  var target = new Date(el.getAttribute('data-target')).getTime();
  var label = el.getAttribute('data-label') || 'Anpfiff in';

  function update() {
    var now = Date.now();
    var diff = target - now;

    if (diff <= 0) {
      el.style.display = 'none';
      return;
    }

    var days = Math.floor(diff / (1000 * 60 * 60 * 24));
    var hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    var minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

    var parts = [];
    if (days > 0) parts.push(days + ' Tagen');
    if (hours > 0) parts.push(hours + ' Stunden');
    if (minutes > 0) parts.push(minutes + ' Minuten');

    var timeText = '';
    if (parts.length === 3) {
      timeText = parts[0] + ', ' + parts[1] + ' und ' + parts[2];
    } else if (parts.length === 2) {
      timeText = parts[0] + ' und ' + parts[1];
    } else if (parts.length === 1) {
      timeText = parts[0];
    }

    if (timeText) {
      el.innerHTML = label + ' <span class="hero__countdown-time">' + timeText + '</span>';
    } else {
      el.style.display = 'none';
    }

    setTimeout(update, 60000);
  }

  update();
})();
