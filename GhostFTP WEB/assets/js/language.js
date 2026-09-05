import { api } from './api.js';

const selector = document.querySelector('#languageSelect');

if (selector) {
    selector.addEventListener('change', async () => {
        const previous = document.querySelector('meta[name="ghostftp-language"]')?.content || 'en';
        selector.disabled = true;
        try {
            const result = await api('me');
            const current = result.preferences && typeof result.preferences === 'object' ? result.preferences : {};
            const preferences = { ...current, language: selector.value };
            await api('save_preferences', { preferences: JSON.stringify(preferences) });
            window.location.reload();
        } catch (error) {
            selector.value = previous;
            window.alert(error?.message || 'The language preference could not be saved.');
            selector.disabled = false;
        }
    });
}
