const state = { users: [], roles: [], permissions: [], methods: [] };

async function request(path, options = {}) {
  const res = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function admin(method, path, body) {
  return request("/api/admin/proxy", {
    method: "POST",
    body: JSON.stringify({ method, path, body: body || null }),
  });
}

function text(value) {
  return String(value || "");
}

function option(value, label, selected) {
  return `<option value="${text(value)}"${value === selected ? " selected" : ""}>${text(label || value)}</option>`;
}

async function load() {
  const data = await request("/api/admin/bootstrap");
  Object.assign(state, data);
  renderUsers();
  renderRBAC();
}

function renderUsers() {
  const tbody = document.querySelector("[data-users]");
  if (!tbody) return;
  const roles = state.roles.map((role) => role.name);
  document.querySelectorAll("[data-role-options]").forEach((select) => {
    select.innerHTML = roles.map((role) => option(role, role)).join("");
  });
  tbody.innerHTML = state.users.map((user) => `
    <tr>
      <td>${text(user.name)}</td>
      <td>${text(user.email)}</td>
      <td>${text(user.company)}</td>
      <td><span class="chip">${text(user.role)}</span></td>
      <td>
        <div class="row-actions">
          <button class="secondary" type="button" data-edit-user="${user.id}">Edit</button>
          <button class="secondary" type="button" data-reset-password="${user.id}">Password</button>
        </div>
      </td>
    </tr>`).join("") || `<tr><td colspan="5">No users found.</td></tr>`;
}

function renderRBAC() {
  const roleList = document.querySelector("[data-roles]");
  const permissionRows = document.querySelector("[data-permissions]");
  if (!roleList || !permissionRows) return;
  document.querySelectorAll("[data-role-id-options]").forEach((select) => {
    select.innerHTML = state.roles.map((role) => option(role.id, role.name)).join("");
  });
  document.querySelectorAll("[data-method-options]").forEach((select) => {
    select.innerHTML = state.methods.map((method) => option(method, method)).join("");
  });
  roleList.innerHTML = state.roles.map((role) => `
    <div class="list-item">
      <div><strong>${text(role.name)}</strong><br><span class="muted">${text(role.description)}</span></div>
      ${["admin", "agent", "customer"].includes(role.name) ? "" : `<button class="danger" type="button" data-delete-role="${role.id}">Delete</button>`}
    </div>`).join("");
  permissionRows.innerHTML = state.permissions.map((permission) => `
    <tr>
      <td><span class="chip">${text(permission.role)}</span></td>
      <td>${text(permission.grpc_method)}</td>
      <td>${text(permission.description)}</td>
      <td><div class="row-actions"><button class="danger" type="button" data-delete-permission="${permission.id}">Remove</button></div></td>
    </tr>`).join("") || `<tr><td colspan="4">No permissions found.</td></tr>`;
}

function openUserForm(user = {}) {
  const panel = document.querySelector("[data-user-form-panel]");
  const form = document.querySelector("[data-user-form]");
  if (!panel || !form) return;
  panel.hidden = false;
  form.elements.id.value = user.id || "";
  form.elements.name.value = user.name || "";
  form.elements.company.value = user.company || "";
  form.elements.email.value = user.email || "";
  form.elements.role.value = user.role || "customer";
  form.elements.password.value = "";
  document.querySelector("[data-user-form-title]").textContent = user.id ? "Edit user" : "Create user";
}

function closeUserForm() {
  const panel = document.querySelector("[data-user-form-panel]");
  if (panel) panel.hidden = true;
}

document.addEventListener("submit", async (event) => {
  const login = event.target.closest("[data-login-form]");
  if (login) {
    event.preventDefault();
    const message = login.querySelector("[data-message]");
    try {
      await request("/api/login", { method: "POST", body: JSON.stringify({ email: login.email.value, password: login.password.value }) });
      location.href = "/admin/users";
    } catch (err) {
      message.textContent = err.message;
    }
  }

  const userForm = event.target.closest("[data-user-form]");
  if (userForm) {
    event.preventDefault();
    const id = userForm.elements.id.value;
    const body = { name: userForm.elements.name.value, company: userForm.elements.company.value, email: userForm.elements.email.value, role: userForm.elements.role.value };
    if (userForm.elements.password.value) body.password = userForm.elements.password.value;
    await admin(id ? "PATCH" : "POST", id ? `/api/admin/users/${id}` : "/api/admin/users", body);
    if (id && userForm.elements.password.value) await admin("PATCH", `/api/admin/users/${id}/password`, { password: userForm.elements.password.value });
    closeUserForm();
    await load();
  }

  const roleForm = event.target.closest("[data-role-form]");
  if (roleForm) {
    event.preventDefault();
    await admin("POST", "/api/admin/roles", { name: roleForm.elements.name.value, description: roleForm.elements.description.value });
    roleForm.reset();
    await load();
  }

  const permissionForm = event.target.closest("[data-permission-form]");
  if (permissionForm) {
    event.preventDefault();
    await admin("POST", "/api/admin/role-permissions", {
      role_id: permissionForm.elements.role_id.value,
      grpc_method: permissionForm.elements.grpc_method.value,
      description: permissionForm.elements.description.value,
    });
    permissionForm.reset();
    await load();
  }
});

document.addEventListener("click", async (event) => {
  if (event.target.matches("[data-logout]")) {
    await request("/api/logout", { method: "POST", body: "{}" });
    location.href = "/login";
  }
  if (event.target.matches("[data-open-user]")) openUserForm();
  if (event.target.matches("[data-cancel-user]")) closeUserForm();
  if (event.target.matches("[data-edit-user]")) {
    openUserForm(state.users.find((user) => user.id === event.target.dataset.editUser));
  }
  if (event.target.matches("[data-reset-password]")) {
    const password = prompt("New password");
    if (password) await admin("PATCH", `/api/admin/users/${event.target.dataset.resetPassword}/password`, { password });
  }
  if (event.target.matches("[data-delete-role]")) {
    await admin("DELETE", `/api/admin/roles/${event.target.dataset.deleteRole}`);
    await load();
  }
  if (event.target.matches("[data-delete-permission]")) {
    await admin("DELETE", `/api/admin/role-permissions/${event.target.dataset.deletePermission}`);
    await load();
  }
  if (event.target.matches("[data-refresh-cache]")) {
    await admin("POST", "/api/admin/rbac/cache/refresh");
    event.target.textContent = "Cache refreshed";
    setTimeout(() => { event.target.textContent = "Refresh cache"; }, 1400);
  }
});

if (document.body.dataset.page) {
  load().catch((err) => {
    document.body.insertAdjacentHTML("afterbegin", `<div class="panel message">${err.message}</div>`);
  });
}
