(() => {

    const navbar = document.getElementById('navbar');

    const darkModeSwitch = document.getElementById('dark-mode');
    darkModeSwitch.checked = darkmode.getSavedColorScheme() === 'dark';
    darkModeSwitch.onchange = () => {
        darkmode.setDarkMode(darkModeSwitch.checked, true);
        applyDarkTheme(darkModeSwitch.checked);
    };
    applyDarkTheme(darkModeSwitch.checked);

    function applyDarkTheme(value) {
        if (value) {
            navbar.classList.add('navbar-dark', 'bg-dark');
            navbar.classList.remove('navbar-light', 'bg-light');
        } else {
            navbar.classList.add('navbar-light', 'bg-light');
            navbar.classList.remove('navbar-dark', 'bg-dark');
        }
    }
})();
