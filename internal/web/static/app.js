"use strict";

// Minimal vanilla dashboard. Talks to the same-origin REST API; the session
// cookie is sent automatically. No build step.

const $ = (sel) => document.querySelector(sel);

function toast(msg, isErr) {
  const t = $("#toast");
  t.textContent = msg;
  t.classList.toggle("err", !!isErr);
  t.classList.add("show");
  setTimeout(() => t.classList.remove("show"), 2500);
}

async function api(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

function fieldInput(field, value) {
  const wrap = document.createElement("div");
  wrap.className = "field";

  const label = document.createElement("label");
  label.textContent = field.label;
  label.htmlFor = `f-${field.key}`;
  wrap.appendChild(label);

  let input;
  if (field.type === "bool") {
    input = document.createElement("input");
    input.type = "checkbox";
    input.checked = !!value;
  } else if (field.type === "int") {
    input = document.createElement("input");
    input.type = "number";
    if (field.min || field.max) { input.min = field.min; input.max = field.max; }
    input.value = value ?? 0;
  } else {
    // string, channel, role -> text
    input = document.createElement("input");
    input.type = "text";
    input.value = value ?? "";
    if (field.type === "channel") input.placeholder = "Channel ID (blank = off)";
    if (field.type === "role") input.placeholder = "Role ID (blank = off)";
  }
  input.id = `f-${field.key}`;
  input.dataset.key = field.key;
  input.dataset.type = field.type;
  wrap.appendChild(input);

  if (field.help) {
    const h = document.createElement("div");
    h.className = "help";
    h.textContent = field.help;
    wrap.appendChild(h);
  }
  return wrap;
}

function readValue(input) {
  if (input.dataset.type === "bool") return input.checked;
  if (input.dataset.type === "int") return Number(input.value);
  return input.value.trim();
}

function renderModule(guildID, mod) {
  const card = document.createElement("div");
  card.className = "card";

  const h = document.createElement("h2");
  h.textContent = mod.title;
  card.appendChild(h);

  const inputs = [];
  for (const field of mod.fields) {
    const wrap = fieldInput(field, mod.values[field.key]);
    card.appendChild(wrap);
    inputs.push(wrap.querySelector("input"));
  }

  const save = document.createElement("button");
  save.textContent = "Save";
  save.onclick = async () => {
    const patch = {};
    for (const inp of inputs) patch[inp.dataset.key] = readValue(inp);
    save.disabled = true;
    try {
      const updated = await api(
        "PATCH",
        `/api/guilds/${guildID}/modules/${mod.module}`,
        patch
      );
      if (updated && updated.values) mod.values = updated.values;
      toast(`${mod.title} saved`);
    } catch (e) {
      toast(e.message, true);
    } finally {
      save.disabled = false;
    }
  };
  card.appendChild(save);
  return card;
}

async function loadModules(guildID) {
  const container = $("#modules");
  container.innerHTML = "<p class='muted'>Loading…</p>";
  try {
    const mods = await api("GET", `/api/guilds/${guildID}/modules`);
    container.innerHTML = "";
    if (!mods.length) {
      container.innerHTML = "<p class='muted'>No configurable modules.</p>";
      return;
    }
    for (const mod of mods) container.appendChild(renderModule(guildID, mod));
  } catch (e) {
    container.innerHTML = `<p class='muted'>${e.message}</p>`;
  }
}

function renderUser(me) {
  const area = $("#user-area");
  area.innerHTML = "";
  const span = document.createElement("span");
  span.className = "muted";
  span.textContent = me.username + "  ";
  const out = document.createElement("button");
  out.className = "secondary";
  out.textContent = "Logout";
  out.onclick = async () => { await api("POST", "/auth/logout"); location.reload(); };
  area.appendChild(span);
  area.appendChild(out);
}

async function init() {
  let me;
  try {
    me = await api("GET", "/api/me");
  } catch {
    $("#login-view").hidden = false;
    $("#app-view").hidden = true;
    return;
  }
  $("#login-view").hidden = true;
  $("#app-view").hidden = false;
  renderUser(me);

  const sel = $("#guild");
  sel.innerHTML = "";
  if (!me.guilds || !me.guilds.length) {
    $("#modules").innerHTML =
      "<p class='muted'>No servers where you can manage the bot.</p>";
    return;
  }
  for (const g of me.guilds) {
    const opt = document.createElement("option");
    opt.value = g.id;
    opt.textContent = g.name;
    sel.appendChild(opt);
  }
  sel.onchange = () => loadModules(sel.value);
  loadModules(sel.value);
}

init();
