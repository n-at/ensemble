(() => {
    const runningStatus = document.getElementById("running-status");
    const url = runningStatus.getAttribute('data-status-url');

    setInterval(() => {
        fetch(url).then(data => {
            return data.json();
        }).then(data => {
            if (data != 1) {
                location.reload();
            }
        })
    }, 1500);

})();
