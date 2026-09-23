(() => {
  const storageKey = "finish-first:completed-missions";
  const allSlugs = [...document.querySelectorAll("[data-mission-link]")].map((node) => node.dataset.missionLink);
  const readCompleted = () => {
    try { return new Set(JSON.parse(localStorage.getItem(storageKey) || "[]")); }
    catch { return new Set(); }
  };
  const writeCompleted = (completed) => localStorage.setItem(storageKey, JSON.stringify([...completed]));

  function renderProgress() {
    const completed = readCompleted();
    document.querySelectorAll("[data-progress-summary]").forEach((node) => {
      node.textContent = `${completed.size} of ${allSlugs.length} exercises`;
    });
    document.querySelectorAll("[data-mission-link]").forEach((node) => node.classList.toggle("is-complete", completed.has(node.dataset.missionLink)));
    document.querySelectorAll("[data-mission-card]").forEach((node) => {
      const done = completed.has(node.dataset.missionCard);
      node.classList.toggle("is-complete", done);
      const status = node.querySelector(".shelf-card__status");
      if (status) status.textContent = done ? "Exercise done" : "Unread";
    });
    document.querySelectorAll("[data-mission-action]").forEach((node) => {
      const done = completed.has(node.dataset.missionAction);
      node.classList.toggle("is-complete", done);
      const button = node.querySelector("button");
      if (button) {
        button.textContent = done ? "Exercise completed ✓" : "Mark exercise complete";
        button.setAttribute("aria-pressed", String(done));
      }
    });
  }

  document.querySelectorAll("[data-complete-mission]").forEach((button) => button.addEventListener("click", () => {
    const completed = readCompleted();
    const slug = button.dataset.completeMission;
    completed.has(slug) ? completed.delete(slug) : completed.add(slug);
    writeCompleted(completed);
    renderProgress();
  }));

  const menuButton = document.querySelector("[data-menu-button]");
  const navigation = document.querySelector("[data-book-nav]");
  if (menuButton && navigation) {
    menuButton.addEventListener("click", () => {
      const open = navigation.classList.toggle("is-open");
      menuButton.setAttribute("aria-expanded", String(open));
    });
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && navigation.classList.contains("is-open")) {
        navigation.classList.remove("is-open");
        menuButton.setAttribute("aria-expanded", "false");
        menuButton.focus();
      }
    });
  }

  const progress = document.querySelector(".reading-progress span");
  if (progress && document.body.classList.contains("article-view")) {
    const updateReadingProgress = () => {
      const maximum = document.documentElement.scrollHeight - innerHeight;
      const ratio = maximum > 0 ? Math.min(1, scrollY / maximum) : 0;
      progress.style.width = `${ratio * 100}%`;
    };
    addEventListener("scroll", updateReadingProgress, { passive: true });
    updateReadingProgress();
  }

  renderProgress();
})();
