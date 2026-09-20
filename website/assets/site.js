(() => {
  const progressBar = document.querySelector(".progress span");
  const progressText = document.getElementById("progressText");
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  if (progressBar instanceof HTMLElement && progressText) {
    let value = 78;

    const render = () => {
      progressBar.style.width = `${value}%`;
      progressText.textContent = `${value}%`;
    };

    if (reduceMotion) {
      value = 92;
      render();
    } else {
      const timer = window.setInterval(() => {
        value += 1;
        render();
        if (value >= 92) window.clearInterval(timer);
      }, 1100);
    }
  }

  document.querySelectorAll("[data-language-select]").forEach((element) => {
    if (!(element instanceof HTMLSelectElement)) return;

    element.addEventListener("change", () => {
      const destination = element.value.trim();
      if (!destination) return;
      window.location.assign(destination);
    });
  });
})();
