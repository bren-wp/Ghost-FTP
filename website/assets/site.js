(() => {
  document.querySelectorAll("[data-language-select]").forEach((element) => {
    if (!(element instanceof HTMLSelectElement)) return;

    element.addEventListener("change", () => {
      const destination = element.value.trim();
      if (!destination) return;
      window.location.assign(destination);
    });
  });
})();
