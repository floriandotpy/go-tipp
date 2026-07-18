// wrapped.js
// Applies goal-distribution bar widths from data attributes. Widths can't be set
// via inline style attributes because the Content-Security-Policy forbids inline
// styles; setting element.style from an external script is allowed.
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.goal-dist__fill[data-pct]').forEach((el) => {
    const pct = parseFloat(el.dataset.pct);
    el.style.width = (isNaN(pct) ? 0 : pct) + '%';
  });
});
