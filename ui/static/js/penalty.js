(function() {
  var scene = document.querySelector('.penalty-scene');
  if (!scene) return;

  var ball = scene.querySelector('.penalty-scene__ball');
  var goal = scene.querySelector('.penalty-scene__goal');
  var keeper = scene.querySelector('.penalty-scene__keeper');
  if (!ball || !goal || !keeper) return;

  var DEBUG = false;
  var GOAL_PADDING_FACTOR = 0.18;
  var ROUND_SIZE = 5;

  // Score tracking
  var roundResults = []; // 'goal', 'save', 'miss'
  var dotsContainer = scene.querySelector('.penalty-scene__dots');

  function initDots() {
    dotsContainer.innerHTML = '';
    for (var i = 0; i < ROUND_SIZE; i++) {
      var dot = document.createElement('span');
      dot.className = 'penalty-scene__dot';
      dotsContainer.appendChild(dot);
    }
  }

  function updateScore(result) {
    roundResults.push(result);

    // Update dot
    var dots = dotsContainer.querySelectorAll('.penalty-scene__dot');
    var idx = roundResults.length - 1;
    if (dots[idx]) {
      dots[idx].classList.add('penalty-scene__dot--' + result);
    }

    // Reset round after ROUND_SIZE kicks
    if (roundResults.length >= ROUND_SIZE) {
      setTimeout(function() {
        roundResults = [];
        initDots();
      }, 1800);
    }
  }

  initDots();

  function getKeeperVisualBounds() {
    // getBoundingClientRect includes transforms, so this gives the actual visual position
    var sceneRect = scene.getBoundingClientRect();
    var keeperRect = keeper.getBoundingClientRect();
    return {
      left: keeperRect.left - sceneRect.left,
      top: keeperRect.top - sceneRect.top,
      width: keeperRect.width,
      height: keeperRect.height
    };
  }

  function drawDebugBoxes() {
    // Remove old debug boxes
    var old = scene.querySelectorAll('.debug-box');
    old.forEach(function(el) { el.parentNode.removeChild(el); });

    if (!DEBUG) return;

    var sceneRect = scene.getBoundingClientRect();
    var goalRect = goal.getBoundingClientRect();

    var goalLeft = goalRect.left - sceneRect.left;
    var goalTop = goalRect.top - sceneRect.top;
    var goalWidth = goalRect.width;
    var goalHeight = goalRect.height;
    var goalPadding = goalWidth * GOAL_PADDING_FACTOR;

    // Goal hit area (green) — only X matters for detection
    var goalBox = document.createElement('div');
    goalBox.className = 'debug-box';
    goalBox.style.cssText = 'position:absolute;pointer-events:none;z-index:50;' +
      'border:2px dashed rgba(91,126,60,0.7);background:rgba(91,126,60,0.08);' +
      'left:' + (goalLeft + goalPadding) + 'px;' +
      'top:' + goalTop + 'px;' +
      'width:' + (goalWidth - goalPadding * 2) + 'px;' +
      'height:' + goalHeight + 'px;';
    scene.appendChild(goalBox);

    // Keeper middle third (red) — uses live visual position
    var kb = getKeeperVisualBounds();
    var keeperMiddleLeft = kb.left + kb.width * 0.05;
    var keeperMiddleWidth = kb.width * 0.90;

    var keeperBox = document.createElement('div');
    keeperBox.className = 'debug-box';
    keeperBox.style.cssText = 'position:absolute;pointer-events:none;z-index:50;' +
      'border:2px dashed rgba(196,69,69,0.7);background:rgba(196,69,69,0.1);' +
      'left:' + keeperMiddleLeft + 'px;' +
      'top:' + kb.top + 'px;' +
      'width:' + keeperMiddleWidth + 'px;' +
      'height:' + kb.height + 'px;';
    scene.appendChild(keeperBox);

    if (DEBUG) {
      requestAnimationFrame(drawDebugBoxes);
    }
  }

  if (DEBUG) {
    requestAnimationFrame(drawDebugBoxes);
  }

  var isAnimating = false;
  var gaugeActive = false;
  var gaugeAngle = 0;
  var gaugeDirection = 1;
  var gaugeSpeed = 0.04;
  var gaugeRAF = null;
  var gauge = null;

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

    var angle = gaugeAngle;
    if (gauge && gauge.parentNode) {
      gauge.parentNode.removeChild(gauge);
    }
    gauge = null;

    kickBall(angle);
  }

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

  var hint = scene.querySelector('.penalty-scene__hint');

  function kickBall(angle) {
    isAnimating = true;
    ball.classList.add('penalty-scene__ball--kicked');
    if (hint) hint.classList.add('penalty-scene__hint--hidden');

    var sceneRect = scene.getBoundingClientRect();
    var ballRect = ball.getBoundingClientRect();
    var goalRect = goal.getBoundingClientRect();

    var startX = ballRect.left + ballRect.width / 2 - sceneRect.left;
    var startY = ballRect.top + ballRect.height / 2 - sceneRect.top;

    var goalLeft = goalRect.left - sceneRect.left;
    var goalTop = goalRect.top - sceneRect.top;
    var goalWidth = goalRect.width;
    var goalHeight = goalRect.height;

    // Target X based on gauge angle
    var goalCenterX = goalLeft + goalWidth / 2;
    var spread = goalWidth * 0.8;
    var targetX = goalCenterX + angle * spread;

    // Target Y
    var targetY = goalTop + goalHeight * (0.3 + Math.random() * 0.4);

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

        // Determine outcome NOW, when ball arrives — keeper position is current
        var kb = getKeeperVisualBounds();
        var keeperMiddleLeft = kb.left + kb.width * 0.05;
        var keeperMiddleRight = kb.left + kb.width * 0.95;

        var goalPadding = goalWidth * GOAL_PADDING_FACTOR;
        var hitsGoalArea = targetX > goalLeft + goalPadding &&
                           targetX < goalLeft + goalWidth - goalPadding;

        var keeperSaves = hitsGoalArea &&
                          targetX > keeperMiddleLeft &&
                          targetX < keeperMiddleRight;

        var isGoal = hitsGoalArea && !keeperSaves;

        // Debug: show where the ball landed (vertical line at targetX)
        if (DEBUG) {
          var sceneHeight = scene.getBoundingClientRect().height;
          var marker = document.createElement('div');
          marker.className = 'debug-target';
          marker.style.cssText = 'position:absolute;pointer-events:none;z-index:55;' +
            'left:' + targetX + 'px;top:0;width:2px;height:' + sceneHeight + 'px;' +
            'background:rgba(255,0,255,0.6);';
          scene.appendChild(marker);
          setTimeout(function() {
            if (marker.parentNode) marker.parentNode.removeChild(marker);
          }, 2000);
        }

        if (isGoal) {
          updateScore('goal');
          showGoal();
        } else if (keeperSaves) {
          updateScore('save');
          showSave();
        } else {
          updateScore('miss');
        }

        setTimeout(function() {
          ball.style.visibility = 'visible';
          ball.classList.remove('penalty-scene__ball--kicked');
          if (hint) hint.classList.remove('penalty-scene__hint--hidden');
          isAnimating = false;
        }, isGoal || keeperSaves ? 1200 : 600);
      }
    }

    requestAnimationFrame(animate);
  }

  function showGoal() {
    goal.classList.add('penalty-scene__goal--shake');
    setTimeout(function() {
      goal.classList.remove('penalty-scene__goal--shake');
    }, 500);

    showMessage('Tor!', 'penalty-scene__tor');
  }

  function showSave() {
    // Pause sway to freeze keeper at current position
    keeper.style.animationPlayState = 'paused';

    // Shake using small offsets from current position
    var shakeFrames = [
      { offset: 0 },
      { transform: 'translateX(-8px)', offset: 0.15 },
      { transform: 'translateX(7px)', offset: 0.3 },
      { transform: 'translateX(-5px)', offset: 0.45 },
      { transform: 'translateX(4px)', offset: 0.6 },
      { transform: 'translateX(-2px)', offset: 0.75 },
      { offset: 1 }
    ];

    var shakeAnim = keeper.animate(shakeFrames, {
      duration: 500,
      easing: 'ease',
      composite: 'add'
    });

    shakeAnim.onfinish = function() {
      keeper.style.animationPlayState = '';
    };

    showMessage('Gehalten!', 'penalty-scene__tor penalty-scene__tor--save');
  }

  function showMessage(text, className) {
    var msg = document.createElement('div');
    msg.className = className;
    msg.textContent = text;
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
