document.addEventListener('submit', (event) => {
    const form = event.target.closest('[data-confirm-delete-user]');
    if (!form) return;
    const label = form.dataset.userLabel || 'ovog korisnika';
    if (!window.confirm(`Obrisati ${label} i sve njegove spremljene GhostFTP podatke? Ova radnja se ne može poništiti.`)) {
        event.preventDefault();
    }
});
