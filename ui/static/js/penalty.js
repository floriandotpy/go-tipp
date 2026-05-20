(function() {
  var scene = document.querySelector('.penalty-scene');
  if (!scene) return;

  var ball = scene.querySelector('.penalty-scene__ball');
  var goal = scene.querySelector('.penalty-scene__goal');
  if (!ball || !goal) return;

  var isAnimating = false;
  var gaugeActive = false;
  var gaugeAngle = 0;
  var gaugeDirection = 1;
  var gaugeSpeed = 0.04;
  var gaugeRAF = null;
  var gauge = null;

  // Create gauge element
  function createGauge() {
    gauge = document.createElement('div');
    gauge.className = 'penalty-gauge';
    gauge.innerHTML = '<div class="penalty-gauge__track"><div class="penalty-gauge__indicator"></div></div>';
    scene.appendChild(gauge);
  }

  function positionGauge() {
    var ballRect = ball.getBoundingClientRect();
    var sceneRect = scene.getBoundingClientRect();
    var cx = ballRect.left + ballRect.width / 2 - sceneRect.left;
    var cy = ballRect.top - sceneRect.top - 20;
    gauge.style.left = cx + 'px';
    gauge.style.top = cy + 'px';
  }

  function startGauge() {
    if (gaugeActive) return;
    gaugeActive = true;
    gaugeAngle = 0;
    gaugeDirection = 1;

    createGauge();
    positionGauge();
    gauge.style.display = 'block';

    function tick() {
      gaugeAngle += gaugeSpeed * gaugeDirection;
      if (gaugeAngle > 1) { gaugeAngle = 1; gaugeDirection = -1; }
      if (gaugeAngle < -1) { gaugeAngle = -1; gaugeDirection = 1; }

      var indicator = gauge.querySelector('.penalty-gauge__indicator');
      // Map -1..1 to 0%..100% position
      var pct = (gaugeAngle + 1) / 2 * 100;
      indicator.style.left = pct + '%';

      gaugeRAF = requestAnimationFrame(tick);
    }
    gaugeRAF = requestAnimationFrame(tick);
  }

  function stopGauge() {
    if (!gaugeActive) return;
    gaugeActive = false;
    cancelAnimationFrame(gaugeRAF);

    var angle = gaugeAngle; // -1 to 1
    if (gauge && gauge.parentNode) {
      gauge.parentNode.removeChild(gauge);
    }
    gauge = null;

    kickBall(angle);
  }

  // Mouse/touch events on ball
  ball.addEventListener('mousedown', function(e) {
    e.preventDefault();
    if (isAnimating) return;
    startGauge();
  });
  ball.addEventListener('touchstart', function(e) {
    e.preventDefault();
    if (isAnimating) return;
    startGauge();
  });

  document.addEventListener('mouseup', function() {
    if (gaugeActive) stopGauge();
  });
  document.addEventListener('touchend', function() {
    if (gaugeActive) stopGauge();
  });

  function kickBall(angle) {
    isAnimating = true;
    ball.classList.add('penalty-scene__ball--kicked');

    var sceneRect = scene.getBoundingClientRect();
    var ballRect = ball.getBoundingClientRect();
    var goalRect = goal.getBoundingClientRect();

    var startX = ballRect.left + ballRect.width / 2 - sceneRect.left;
    var startY = ballRect.top + ballRect.height / 2 - sceneRect.top;

    var goalLeft = goalRect.left - sceneRect.left;
    var goalTop = goalRect.top - sceneRect.top;
    var goalWidth = goalRect.width;
    var goalHeight = goalRect.height;

    // Target X based on gauge angle (-1 to 1), with spread beyond goal
    var goalCenterX = goalLeft + goalWidth / 2;
    var spread = goalWidth * 0.8;
    var targetX = goalCenterX + angle * spread;

    // Target Y: aim at upper-middle of goal with slight randomness
    var targetY = goalTop + goalHeight * (0.3 + Math.random() * 0.4);

    // Determine if it's a goal
    var goalPadding = goalWidth * 0.08;
    var isGoal = targetX > goalLeft + goalPadding &&
                 targetX < goalLeft + goalWidth - goalPadding &&
                 targetY > goalTop + goalPadding &&
                 targetY < goalTop + goalHeight - goalPadding;

    var duration = 600;
    var startTime = null;

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

    ball.style.visibility = 'hidden';

    function animate(timestamp) {
      if (!startTime) startTime = timestamp;
      var elapsed = timestamp - startTime;
      var t = Math.min(elapsed / duration, 1);

      var ease = 1 - Math.pow(1 - t, 2);

      var x = startX + (targetX - startX) * ease;
      var y = startY + (targetY - startY) * ease;

      var arc = -120 * Math.sin(t * Math.PI);
      y += arc;

      var scale = 1 - 0.5 * ease;
      var opacity = 1 - 0.4 * ease;

      flyBall.style.left = x + 'px';
      flyBall.style.top = y + 'px';
      flyBall.style.transform = 'translate(-50%, -50%) scale(' + scale + ')';
      flyBall.style.opacity = opacity;

      if (t < 1) {
        requestAnimationFrame(animate);
      } else {
        scene.removeChild(flyBall);

        if (isGoal) {
          showGoal();
        }

        setTimeout(function() {
          ball.style.visibility = 'visible';
          ball.classList.remove('penalty-scene__ball--kicked');
          isAnimating = false;
        }, isGoal ? 1200 : 600);
      }
    }

    requestAnimationFrame(animate);
  }

  function showGoal() {
    goal.classList.add('penalty-scene__goal--shake');
    setTimeout(function() {
      goal.classList.remove('penalty-scene__goal--shake');
    }, 500);

    var msg = document.createElement('div');
    msg.className = 'penalty-scene__tor';
    msg.textContent = 'Tor!';
    scene.appendChild(msg);

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
