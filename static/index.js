// Fetch the daily tip
fetch('/daily-tip')
  .then(response => response.json())
  .then(data => {
    document.getElementById('vim-tip').innerText = data.tip;
  })
  .catch(error => {
    console.error('Error fetching tip:', error);
  });

// Fetch a random tip on button click
document.getElementById('random-tip-btn').addEventListener('click', () => {
  fetch('/random-tip')
    .then(response => response.json())
    .then(data => {
      document.getElementById('vim-tip').innerText = data.tip;
    })
    .catch(error => {
      console.error('Error fetching random tip:', error);
    });
});

