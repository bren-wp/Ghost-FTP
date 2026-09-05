let deferredInstallPrompt = null;

function isStandalone() {
    return window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true;
}

function isIos() {
    return /iphone|ipad|ipod/i.test(navigator.userAgent) && !window.MSStream;
}

function updateInstallButtons() {
    document.querySelectorAll('[data-install-app]').forEach((button) => {
        button.hidden = isStandalone();
        button.classList.toggle('install-ready', Boolean(deferredInstallPrompt));
    });
}

async function installApp() {
    if (isStandalone()) return;
    if (deferredInstallPrompt) {
        deferredInstallPrompt.prompt();
        await deferredInstallPrompt.userChoice;
        deferredInstallPrompt = null;
        updateInstallButtons();
        return;
    }

    const modal = document.querySelector('#installModal');
    if (modal) {
        modal.classList.remove('hidden');
        const ios = modal.querySelector('[data-ios-install]');
        const generic = modal.querySelector('[data-generic-install]');
        if (ios) ios.hidden = !isIos();
        if (generic) generic.hidden = isIos();
        return;
    }

    if (isIos()) {
        alert('On iPhone or iPad, open Share and choose “Add to Home Screen”.');
    } else {
        alert('In your browser menu, choose “Install app” or “Add to Home Screen”.');
    }
}

window.addEventListener('beforeinstallprompt', (event) => {
    event.preventDefault();
    deferredInstallPrompt = event;
    updateInstallButtons();
});
window.addEventListener('appinstalled', () => {
    deferredInstallPrompt = null;
    updateInstallButtons();
});

document.addEventListener('click', (event) => {
    const install = event.target.closest('[data-install-app]');
    if (install) {
        event.preventDefault();
        installApp();
    }
    const close = event.target.closest('[data-close-install]');
    if (close) {
        document.querySelector('#installModal')?.classList.add('hidden');
    }
});

if ('serviceWorker' in navigator && window.isSecureContext) {
    window.addEventListener('load', () => {
        const swUrl = document.querySelector('meta[name="service-worker-url"]')?.content || 'service-worker.js';
        navigator.serviceWorker.register(swUrl).catch(() => undefined);
    });
}

updateInstallButtons();
