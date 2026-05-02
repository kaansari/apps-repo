const jsonHeaders = { "Content-Type": "application/json" };

function formData(form) {
  return Object.fromEntries(new FormData(form).entries());
}

function setMessage(form, text, tone = "error") {
  const message = form.querySelector("[data-message]");
  if (!message) return;
  message.textContent = text;
  message.dataset.tone = tone;
}

async function postJSON(url, body) {
  const response = await fetch(url, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "same-origin",
    body: JSON.stringify(body)
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || "Request failed.");
  }
  return payload;
}

document.querySelectorAll("[data-auth-form]").forEach((form) => {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;

    const button = form.querySelector("button[type='submit']");
    button.disabled = true;
    setMessage(form, "Working...", "neutral");

    try {
      await postJSON(form.dataset.endpoint, formData(form));
      window.location.assign(form.dataset.success || "/");
    } catch (error) {
      setMessage(form, error.message);
    } finally {
      button.disabled = false;
    }
  });
});

document.querySelectorAll("[data-password-form]").forEach((form) => {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;

    const data = formData(form);
    if (data.newPassword !== data.confirmPassword) {
      setMessage(form, "New password and confirmation must match.");
      return;
    }

    const button = form.querySelector("button[type='submit']");
    button.disabled = true;
    setMessage(form, "Working...", "neutral");

    try {
      await postJSON("/api/change-password", data);
      setMessage(form, "Password updated.", "success");
      form.reset();
    } catch (error) {
      setMessage(form, error.message);
    } finally {
      button.disabled = false;
    }
  });
});

document.querySelectorAll("[data-profile-form]").forEach((form) => {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;

    const button = form.querySelector("button[type='submit']");
    button.disabled = true;
    setMessage(form, "Saving...", "neutral");

    try {
      const payload = await postJSON("/api/profile", formData(form));
      if (payload.user) {
        ["name", "email", "company"].forEach((field) => {
          if (form.elements[field]) {
            form.elements[field].value = payload.user[field] || "";
          }
        });
      }
      setMessage(form, "Profile updated.", "success");
    } catch (error) {
      setMessage(form, error.message);
    } finally {
      button.disabled = false;
    }
  });
});

const avatarButton = document.querySelector("[data-avatar-button]");
const avatarMenu = document.querySelector("[data-avatar-menu]");

if (avatarButton && avatarMenu) {
  avatarButton.addEventListener("click", () => {
    const open = avatarMenu.hidden;
    avatarMenu.hidden = !open;
    avatarButton.setAttribute("aria-expanded", String(open));
  });

  document.addEventListener("click", (event) => {
    if (!avatarMenu.hidden && !event.target.closest(".avatar-menu")) {
      avatarMenu.hidden = true;
      avatarButton.setAttribute("aria-expanded", "false");
    }
  });
}

document.querySelectorAll("[data-logout]").forEach((button) => {
  button.addEventListener("click", async () => {
    button.disabled = true;
    try {
      await postJSON("/api/logout", {});
    } finally {
      window.location.assign("/login");
    }
  });
});

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/service-worker.js").catch(() => {});
  });
}
