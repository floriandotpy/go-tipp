// script.js

// Function to generate random colors for each user
function stringToHashCode(str) {
    let hash = 0;
    if (str.length === 0) return hash;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = ((hash << 5) - hash) + char;
        hash |= 0; // Convert to 32bit integer
    }
    return hash;
}

function getRandomColor(user_name) {
    const hashCode = stringToHashCode(user_name);
    
    // Use the hash code to generate a color
    const r = (hashCode & 0xFF0000) >> 16;
    const g = (hashCode & 0x00FF00) >> 8;
    const b = hashCode & 0x0000FF;
    
    const color = `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
    
    return color
}

// Function to create chart with the fetched data
function createChart(canvas, data) {
    const matches = data.matches;
    const users = data.users;

    const datasets = users.map(user => {
        return {
            label: user.name,
            data: user.total_points,
            fill: false,
            borderColor: getRandomColor(user.name),
            tension: 0.1
        };
    });

    // Chart configuration
    const config = {
        type: 'line',
        data: {
            labels: matches,
            datasets: datasets
        },
        options: {
            responsive: true,
            scales: {
                x: {
                    title: {
                        display: true,
                        text: 'Match'
                    }
                },
                y: {
                    title: {
                        display: true,
                        text: 'Points'
                    },
                    beginAtZero: true
                }
            }
        }
    };

    // Render the chart
    const ctx = canvas.getContext('2d');
    new Chart(ctx, config);

}


// Function to fetch data and create chart for a single canvas element
function fetchDataAndCreateChart(canvas) {
    const groupIds = canvas.dataset.chart;
    let url = '/scores.json?groups=' + groupIds;
    
    fetch(url)
        .then(response => response.json())
        .then(data => {
            createChart(canvas, data);
        })
        .catch(error => {
            console.error(`Error fetching data for group ${groupId}:`, error);
        });
}

// Select all canvas elements with data-chart attribute and create charts
document.addEventListener('DOMContentLoaded', () => {
    const chartCanvases = document.querySelectorAll('canvas[data-chart]');
    chartCanvases.forEach(canvas => {
        fetchDataAndCreateChart(canvas);
    });
});




