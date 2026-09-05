document.addEventListener('submit', (event) => {
    const form = event.target.closest('[data-confirm-delete-user]');
    if (!form) return;
    const label = form.dataset.userLabel || 'this user';
    if (!window.confirm(`Delete ${label} and all of their saved Ghost FTP data? This action cannot be undone.`)) {
        event.preventDefault();
    }
});
