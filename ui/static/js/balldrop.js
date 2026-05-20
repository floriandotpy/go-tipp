(function() {
  const canvas = document.getElementById('ballDropCanvas');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  const wrapper = canvas.parentElement;

  const BALL_SIZE = 64;
  const NUM_BALLS = 7;

  let ballImg = new Image();
  ballImg.src = '/static/img/ball.png';

  function resize() {
    canvas.width = wrapper.clientWidth;
    canvas.height = wrapper.clientHeight;
  }
  resize();
  window.addEventListener('resize', resize);

  function draw(time) {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const cx = canvas.width / 2;
    const cy = canvas.height / 2;
    // Radii for the figure-8 (lemniscate) path
    const rx = Math.min(canvas.width * 0.35, 200);
    const ry = Math.min(canvas.height * 0.35, 120);

    for (let i = 0; i < NUM_BALLS; i++) {
      // Evenly space balls along the path with time offset
      const t = (time * 0.001) + (i * Math.PI * 2 / NUM_BALLS);

      // Lemniscate of Bernoulli parametric form gives a figure-8
      const x = cx + rx * Math.sin(t);
      const y = cy + ry * Math.sin(t * 2) * 0.5;

      // Simulate depth with scale (3D feel from the sin(t) z-component)
      const z = Math.cos(t);
      const scale = 0.6 + 0.4 * ((z + 1) / 2); // range 0.6 to 1.0
      const size = BALL_SIZE * scale;

      // Slight rotation based on movement
      const angle = t * 1.5;

      ctx.save();
      ctx.globalAlpha = 0.7 + 0.3 * ((z + 1) / 2); // slightly fade distant balls
      ctx.translate(x, y);
      ctx.rotate(angle);
      ctx.drawImage(ballImg, -size / 2, -size / 2, size, size);
      ctx.restore();
    }

    requestAnimationFrame(draw);
  }

  ballImg.onload = function() {
    requestAnimationFrame(draw);
  };
})();
