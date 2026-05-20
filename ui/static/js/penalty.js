(function() {
  var scene = document.querySelector('.penalty-scene');
  if (!scene) return;

  var ball = scene.querySelector('.penalty-scene__ball');
  var goal = scene.querySelector('.penalty-scene__goal');
  var keeper = scene.querySelector('.penalty-scene__keeper');
  if (!ball || !goal) return;

  var isAnimating = false;

  ball.addEventListener('click', function() {
    if (isAnimating) return;
    isAnimating = true;

    // Stop pulsate during flight
    ball.classList.add('penalty-scene__ball--kicked');

    // Get positions
    var sceneRect = scene.getBoundingClientRect();
    var ballRect = ball.getBoundingClientRect();
    var goalRect = goal.getBoundingClientRect();

    // Ball start (center of ball relative to scene)
    var startX = ballRect.left + ballRect.width / 2 - sceneRect.left;
    var startY = ballRect.top + ballRect.height / 2 - sceneRect.top;

    // Target: random point within the goal area (with some margin)
    var goalLeft = goalRect.left - sceneRect.left;
    var goalTop = goalRect.top - sceneRect.top;
    var goalWidth = goalRect.width;
    var goalHeight = goalRect.height;

    // Random horizontal angle: target anywhere across the width, with some chance to miss
    var spread = 1.4; // >1 means can go outside goal
    var targetX = goalLeft + goalWidth * 0.5 + (Math.random() - 0.5) * goalWidth * spread;
    var targetY = goalTop + goalHeight * 0.5 + (Math.random() - 0.5) * goalHeight * 0.6;

    // Determine if it's a goal (target lands within goal bounds with some padding)
    var goalPadding = goalWidth * 0.08;
    var isGoal = targetX > goalLeft + goalPadding &&
                 targetX < goalLeft + goalWidth - goalPadding &&
                 targetY > goalTop + goalPadding &&
                 targetY < goalTop + goalHeight - goalPadding;

    // Animation params
    var duration = 600;
    var startTime = null;

    // Create a clone for the flying ball
    var flyBall = ball.cloneNode(true);
    flyBall.classList.add('penalty-scene__ball--flying');
    flyBall.style.position = 'absolute';
    flyBall.style.left = startX + 'px';
    flyBall.style.top = startY + 'px';
    flyBall.style.transform = 'translate(-50%, -50%)';
    flyBall.style.margin = '0';
    flyBall.style.animation = 'none';
    flyBall.style.pointerEvents = 'none';
    flyBall.style.zIndex = '10';
    scene.appendChild(flyBall);

    // Hide original ball during flight
    ball.style.visibility = 'hidden';

    function animate(timestamp) {
      if (!startTime) startTime = timestamp;
      var elapsed = timestamp - startTime;
      var t = Math.min(elapsed / duration, 1);

      // Ease out
      var ease = 1 - Math.pow(1 - t, 2);

      // Position along path
      var x = startX + (targetX - startX) * ease;
      var y = startY + (targetY - startY) * ease;

      // Arc: ball goes up in the middle of the flight
      var arc = -120 * Math.sin(t * Math.PI);
      y += arc;

      // Scale down and fade as it "goes into distance"
      var scale = 1 - 0.5 * ease;
      var opacity = 1 - 0.4 * ease;

      flyBall.style.left = x + 'px';
      flyBall.style.top = y + 'px';
      flyBall.style.transform = 'translate(-50%, -50%) scale(' + scale + ')';
      flyBall.style.opacity = opacity;

      if (t < 1) {
        requestAnimationFrame(animate);
      } else {
        // Animation done
        scene.removeChild(flyBall);

        if (isGoal) {
          showGoal();
        }

        // Reset ball
        setTimeout(function() {
          ball.style.visibility = 'visible';
          ball.classList.remove('penalty-scene__ball--kicked');
          isAnimating = false;
        }, isGoal ? 1200 : 600);
      }
    }

    requestAnimationFrame(animate);
  });

  function showGoal() {
    // Shake the goal
    goal.classList.add('penalty-scene__goal--shake');
    setTimeout(function() {
      goal.classList.remove('penalty-scene__goal--shake');
    }, 500);

    // Floating "Tor!" message
    var msg = document.createElement('div');
    msg.className = 'penalty-scene__tor';
    msg.textContent = 'Tor!';
    scene.appendChild(msg);

    // Trigger animation after append
    requestAnimationFrame(function() {
      msg.classList.add('penalty-scene__tor--visible');
    });

    setTimeout(function() {
      msg.classList.add('penalty-scene__tor--fade');
    }, 100);

    setTimeout(function() {
      if (msg.parentNode) {
        msg.parentNode.removeChild(msg);
      }
    }, 1500);
  }
})();
