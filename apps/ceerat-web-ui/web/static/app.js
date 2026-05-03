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

async function getJSON(url) {
  const response = await fetch(url, { credentials: "same-origin" });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || "Request failed.");
  }
  return payload;
}

function fillSelect(select, items, labelFor, emptyText) {
  if (!select) return;
  select.replaceChildren();
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = emptyText;
  select.append(empty);

  items.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.id || "";
    option.textContent = labelFor(item);
    select.append(option);
  });
}

function customerName(customer) {
  const name = [customer.firstName, customer.lastName].filter(Boolean).join(" ").trim();
  return name || customer.email || "Unnamed customer";
}

function customerAddress(customer) {
  const address = customer.address || {};
  return [address.line1, address.city, address.state, address.postalCode].filter(Boolean).join(", ");
}

function formatMoney(value) {
  const amount = Number(value || 0);
  return amount ? amount.toLocaleString(undefined, { style: "currency", currency: "USD" }) : "";
}

function setButtonBusy(button, busy) {
  if (button) button.disabled = busy;
}

function showCustomerPanel() {
  const panel = document.querySelector("[data-customer-panel]");
  if (panel) panel.hidden = false;
}

function hideCustomerPanel() {
  const panel = document.querySelector("[data-customer-panel]");
  if (panel) panel.hidden = true;
}

function showAssignServiceForm() {
  const form = document.querySelector("[data-assign-service-form]");
  const toggle = document.querySelector("[data-toggle-assign-service]");
  if (form) form.hidden = false;
  if (toggle) {
    toggle.textContent = "-";
    toggle.setAttribute("aria-expanded", "true");
  }
}

function hideAssignServiceForm() {
  const form = document.querySelector("[data-assign-service-form]");
  const toggle = document.querySelector("[data-toggle-assign-service]");
  if (form) {
    form.reset();
    setMessage(form, "");
    form.hidden = true;
  }
  if (toggle) {
    toggle.textContent = "+";
    toggle.setAttribute("aria-expanded", "false");
  }
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

const dashboard = document.querySelector("[data-dashboard]");
let dashboardState = { customers: [], services: [], customerServices: [] };

async function loadDashboard() {
  if (!dashboard) return;
  try {
    dashboardState = await getJSON("/api/dashboard");
    renderDashboard();
  } catch (error) {
    const customersList = document.querySelector("[data-customers-list]");
    const customerServicesList = document.querySelector("[data-customer-services-list]");
    if (customersList) customersList.innerHTML = `<tr><td colspan="4">${error.message}</td></tr>`;
    if (customerServicesList) customerServicesList.innerHTML = `<tr><td colspan="5">${error.message}</td></tr>`;
  }
}

function renderDashboard() {
  renderCustomers();
  renderOptions();
  renderCustomerServices();
}

function renderCustomers() {
  const list = document.querySelector("[data-customers-list]");
  if (!list) return;
  list.replaceChildren();

  if (!dashboardState.customers.length) {
    const row = document.createElement("tr");
    row.innerHTML = `<td colspan="4">No customers yet.</td>`;
    list.append(row);
    return;
  }

  dashboardState.customers.forEach((customer) => {
    const row = document.createElement("tr");
    const name = document.createElement("td");
    name.textContent = customerName(customer);
    const contact = document.createElement("td");
    contact.textContent = [customer.email, customer.phone].filter(Boolean).join(" / ") || "No contact";
    const address = document.createElement("td");
    address.textContent = customerAddress(customer) || "No address";
    const actions = document.createElement("td");
    const edit = document.createElement("button");
    edit.type = "button";
    edit.className = "table-action";
    edit.textContent = "Edit";
    edit.addEventListener("click", () => populateCustomerForm(customer));
    actions.append(edit);
    row.append(name, contact, address, actions);
    list.append(row);
  });
}

function renderOptions() {
  fillSelect(
    document.querySelector("[data-customer-options]"),
    dashboardState.customers,
    customerName,
    "Select customer"
  );
  fillSelect(
    document.querySelector("[data-service-options]"),
    dashboardState.services,
    (service) => `${service.name}${service.category ? ` - ${service.category}` : ""}${service.price ? ` (${formatMoney(service.price)})` : ""}`,
    "Select service"
  );
}

function renderCustomerServices() {
  const list = document.querySelector("[data-customer-services-list]");
  if (!list) return;
  list.replaceChildren();

  if (!dashboardState.customerServices.length) {
    const row = document.createElement("tr");
    row.innerHTML = `<td colspan="5">No services assigned yet.</td>`;
    list.append(row);
    return;
  }

  dashboardState.customerServices.forEach((item) => {
    const row = document.createElement("tr");
    const customer = document.createElement("td");
    customer.textContent = customerName(item.customer || {}) || item.customerId;
    const service = document.createElement("td");
    service.textContent = item.service?.name || item.serviceId;
    const status = document.createElement("td");
    const statusInput = document.createElement("input");
    statusInput.value = item.status || "";
    statusInput.name = "status";
    status.append(statusInput);
    const orderedAt = document.createElement("td");
    const dateInput = document.createElement("input");
    dateInput.type = "date";
    dateInput.name = "orderedAt";
    dateInput.value = (item.orderedAt || "").slice(0, 10);
    orderedAt.append(dateInput);
    const actions = document.createElement("td");
    const save = document.createElement("button");
    save.type = "button";
    save.className = "table-action";
    save.textContent = "Save";
    save.addEventListener("click", async () => {
      setButtonBusy(save, true);
      try {
        await postJSON("/api/customer-services/update", {
          id: item.id,
          customerId: item.customerId,
          serviceId: item.serviceId,
          status: statusInput.value,
          orderedAt: dateInput.value
        });
        await loadDashboard();
      } catch (error) {
        window.alert(error.message);
      } finally {
        setButtonBusy(save, false);
      }
    });
    actions.append(save);
    row.append(customer, service, status, orderedAt, actions);
    list.append(row);
  });
}

function populateCustomerForm(customer) {
  const form = document.querySelector("[data-customer-form]");
  if (!form) return;
  showCustomerPanel();
  const address = customer.address || {};
  const values = {
    id: customer.id || "",
    firstName: customer.firstName || "",
    lastName: customer.lastName || "",
    email: customer.email || "",
    phone: customer.phone || "",
    line1: address.line1 || "",
    line2: address.line2 || "",
    city: address.city || "",
    state: address.state || "",
    country: address.country || "",
    postalCode: address.postalCode || ""
  };
  Object.entries(values).forEach(([key, value]) => {
    if (form.elements[key]) form.elements[key].value = value;
  });
  const title = document.querySelector("[data-customer-form-title]");
  if (title) title.textContent = "Update customer";
  form.scrollIntoView({ behavior: "smooth", block: "start" });
}

function resetCustomerForm() {
  const form = document.querySelector("[data-customer-form]");
  if (!form) return;
  form.reset();
  if (form.elements.id) form.elements.id.value = "";
  const title = document.querySelector("[data-customer-form-title]");
  if (title) title.textContent = "Create customer";
  setMessage(form, "");
  hideCustomerPanel();
}

document.querySelectorAll("[data-customer-form]").forEach((form) => {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;

    const data = formData(form);
    const endpoint = data.id ? "/api/customers/update" : "/api/customers";
    const button = form.querySelector("button[type='submit']");
    setButtonBusy(button, true);
    setMessage(form, "Saving...", "neutral");

    try {
      await postJSON(endpoint, data);
      setMessage(form, "Customer saved.", "success");
      resetCustomerForm();
      await loadDashboard();
    } catch (error) {
      setMessage(form, error.message);
    } finally {
      setButtonBusy(button, false);
    }
  });
});

document.querySelectorAll("[data-reset-customer]").forEach((button) => {
  button.addEventListener("click", resetCustomerForm);
});

document.querySelectorAll("[data-new-customer]").forEach((button) => {
  button.addEventListener("click", () => {
    resetCustomerForm();
    showCustomerPanel();
    const form = document.querySelector("[data-customer-form]");
    if (form) form.scrollIntoView({ behavior: "smooth", block: "start" });
  });
});

document.querySelectorAll("[data-toggle-assign-service]").forEach((button) => {
  button.addEventListener("click", () => {
    const form = document.querySelector("[data-assign-service-form]");
    if (form?.hidden) {
      showAssignServiceForm();
    } else {
      hideAssignServiceForm();
    }
  });
});

document.querySelectorAll("[data-cancel-assign-service]").forEach((button) => {
  button.addEventListener("click", hideAssignServiceForm);
});

document.querySelectorAll("[data-assign-service-form]").forEach((form) => {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;

    const button = form.querySelector("button[type='submit']");
    setButtonBusy(button, true);
    setMessage(form, "Assigning...", "neutral");

    try {
      await postJSON("/api/customer-services", formData(form));
      setMessage(form, "Service assigned.", "success");
      hideAssignServiceForm();
      await loadDashboard();
    } catch (error) {
      setMessage(form, error.message);
    } finally {
      setButtonBusy(button, false);
    }
  });
});

loadDashboard();

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
