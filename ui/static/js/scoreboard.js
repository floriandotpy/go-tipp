// scoreboard.js

// Curated color palette — visually distinct, modern, accessible on white
const COLORS = [
  '#6366f1', // indigo
  '#f59e0b', // amber
  '#10b981', // emerald
  '#ef4444', // red
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
  '#06b6d4', // cyan
  '#84cc16', // lime
  '#a855f7', // purple
  '#e11d48', // rose
  '#0ea5e9', // sky
  '#22c55e', // green
  '#eab308', // yellow
  '#78716c', // stone
  '#2dd4bf', // teal-light
  '#c026d3', // fuchsia
  '#dc2626', // red-dark
];

function getColor(index) {
  return COLORS[index % COLORS.length];
}

// Compute rank at each match index from cumulative totals.
// Returns an object { userName: [rank1, rank2, ...] }
function computeRankHistory(users) {
  if (users.length === 0) return {};

  const matchCount = users[0].total_points.length;
  const rankHistory = {};
  users.forEach(u => { rankHistory[u.name] = []; });

  for (let i = 0; i < matchCount; i++) {
    // Build array of {name, points} at this match index
    const snapshot = users.map(u => ({
      name: u.name,
      points: u.total_points[i],
    }));

    // Sort descending by points
    snapshot.sort((a, b) => b.points - a.points);

    // Assign ranks (handle ties — same points = same rank)
    let rank = 1;
    for (let j = 0; j < snapshot.length; j++) {
      if (j > 0 && snapshot[j].points < snapshot[j - 1].points) {
        rank = j + 1;
      }
      rankHistory[snapshot[j].name].push(rank);
    }
  }

  return rankHistory;
}

// Slice data arrays to the last N entries
function sliceData(arr, limit) {
  if (!limit || limit >= arr.length) return arr;
  return arr.slice(-limit);
}

function createChart(canvas, data, { timeframe, mode }) {
  const users = data.users;
  const matches = data.matches;

  // Sort users by current total points (descending) so legend matches leaderboard
  users.sort((a, b) => {
    const lastA = a.total_points[a.total_points.length - 1] || 0;
    const lastB = b.total_points[b.total_points.length - 1] || 0;
    return lastB - lastA;
  });

  // Determine limit from timeframe
  let limit = null;
  if (timeframe === '10') limit = 10;
  else if (timeframe === '25') limit = 25;
  // 'all' → no limit

  // Compute rank history if needed
  let rankHistory = null;
  if (mode === 'rank') {
    rankHistory = computeRankHistory(users);
  }

  const labels = sliceData(matches, limit);

  const datasets = users.map((user, idx) => {
    let yData;
    if (mode === 'rank') {
      yData = sliceData(rankHistory[user.name], limit);
    } else {
      yData = sliceData(user.total_points, limit);
    }

    return {
      label: user.name,
      data: yData,
      fill: false,
      borderColor: getColor(idx),
      backgroundColor: getColor(idx),
      borderWidth: 1.8,
      pointRadius: 2,
      pointHoverRadius: 5,
      tension: 0.3,
    };
  });

  const config = {
    type: 'line',
    data: {
      labels: labels,
      datasets: datasets,
    },
    options: {
      responsive: true,
      animation: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        legend: {
          position: 'bottom',
          labels: {
            usePointStyle: true,
            pointStyle: 'circle',
            padding: 16,
            font: { size: 12 },
          },
          onClick: (event, legendItem, legend) => {
            const chart = legend.chart;
            const idx = legendItem.datasetIndex;

            // If already highlighted, reset all
            if (chart._highlightedIndex === idx) {
              chart.data.datasets.forEach((dataset, i) => {
                dataset.borderWidth = 1.8;
                dataset.borderColor = getColor(i);
                dataset.pointBackgroundColor = getColor(i);
              });
              chart._highlightedIndex = null;
            } else {
              // Highlight clicked, dim others
              chart.data.datasets.forEach((dataset, i) => {
                dataset.borderWidth = i === idx ? 3 : 1.8;
                dataset.borderColor = i === idx
                  ? getColor(i)
                  : getColor(i) + '26';
                dataset.pointBackgroundColor = i === idx
                  ? getColor(i)
                  : getColor(i) + '26';
              });
              chart._highlightedIndex = idx;
            }
            chart.update('none');
          },
          onHover: (event, legendItem, legend) => {
            const chart = legend.chart;
            if (chart._highlightedIndex != null) return; // don't override tap selection
            chart.data.datasets.forEach((dataset, i) => {
              dataset.borderWidth = i === legendItem.datasetIndex ? 3 : 1.8;
              dataset.borderColor = i === legendItem.datasetIndex
                ? getColor(i)
                : getColor(i) + '26';
              dataset.pointBackgroundColor = i === legendItem.datasetIndex
                ? getColor(i)
                : getColor(i) + '26';
            });
            chart.update('none');
          },
          onLeave: (event, legendItem, legend) => {
            const chart = legend.chart;
            if (chart._highlightedIndex != null) return; // don't override tap selection
            chart.data.datasets.forEach((dataset, i) => {
              dataset.borderWidth = 1.8;
              dataset.borderColor = getColor(i);
              dataset.pointBackgroundColor = getColor(i);
            });
            chart.update('none');
          },
        },
        tooltip: {
          backgroundColor: 'rgba(0,0,0,0.8)',
          titleFont: { size: 12 },
          bodyFont: { size: 11 },
          padding: 10,
          cornerRadius: 6,
          mode: 'index',
          intersect: false,
        },
      },
      scales: {
        x: {
          title: {
            display: true,
            text: 'Spiel',
            font: { size: 12 },
          },
          grid: {
            display: false,
          },
          ticks: {
            font: { size: 11 },
            maxTicksLimit: 15,
          },
        },
        y: {
          title: {
            display: true,
            text: mode === 'rank' ? 'Rang' : 'Punkte',
            font: { size: 12 },
          },
          reverse: mode === 'rank',
          beginAtZero: mode !== 'rank',
          grid: {
            color: 'rgba(0,0,0,0.05)',
          },
          ticks: {
            font: { size: 11 },
            precision: 0,
          },
        },
      },
    },
  };

  const ctx = canvas.getContext('2d');
  return new Chart(ctx, config);
}

// Initialize a chart container with controls
function initChartContainer(container) {
  const canvas = container.querySelector('canvas[data-chart]');
  if (!canvas) return;

  const groupIds = canvas.dataset.chart;
  const url = '/scores.json?groups=' + groupIds;

  // State
  let chartData = null;
  let chartInstance = null;
  let currentTimeframe = '10';
  let currentMode = 'points';

  // Wire up controls
  const timeframeBtns = container.querySelectorAll('[data-timeframe]');
  const modeBtns = container.querySelectorAll('[data-mode]');

  timeframeBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      currentTimeframe = btn.dataset.timeframe;
      timeframeBtns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      renderChart();
    });
  });

  modeBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      currentMode = btn.dataset.mode;
      modeBtns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      renderChart();
    });
  });

  function renderChart() {
    if (!chartData) return;
    if (chartInstance) {
      chartInstance.destroy();
    }
    chartInstance = createChart(canvas, chartData, {
      timeframe: currentTimeframe,
      mode: currentMode,
    });
  }

  // Fetch data and render
  fetch(url)
    .then(response => response.json())
    .then(data => {
      chartData = data;
      renderChart();
    })
    .catch(error => {
      console.error('Error fetching scoreboard data:', error);
    });
}

// Initialize all chart containers on the page
document.addEventListener('DOMContentLoaded', () => {
  const containers = document.querySelectorAll('.chart-container');
  containers.forEach(container => {
    initChartContainer(container);
  });
});
