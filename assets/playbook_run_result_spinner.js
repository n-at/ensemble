(() => {
    const runningStatus = document.getElementById("running-status");
    if (!runningStatus) {
        return;
    }
    const statusUrl = runningStatus.getAttribute('data-status-url');
    const authUrl = "/auth";

    const pollStatus = () => fetch(statusUrl).then(r => r.json());

    const pollAuth = () => fetch(authUrl).then(r => r.ok && r.json() === true);

    const refresh = () => {
        Promise.allSettled([pollStatus(), pollAuth()]).then(results => {
            const [statusResult, authResult] = results;

            if (statusResult.status === "fulfilled" && authResult.status === "fulfilled") {
                const data = statusResult.value;
                if (data !== 1) {
                    if (authResult.value) {
                        location.reload();
                    } else {
                        location.href = "/login";
                    }
                }
            }
        });
    };

    setInterval(refresh, 1500);

})();
