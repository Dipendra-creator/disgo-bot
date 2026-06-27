"use strict";

/* disgo dashboard — vanilla SPA, no build step. Hash router, schema-driven
   config forms, and role/channel pickers populated from the gateway state
   cache. Auth is the same-origin session cookie sent automatically by fetch. */

/* ------------------------------------------------------------------ helpers */

const $ = (sel, root = document) => root.querySelector(sel);

// el builds a DOM node. props: class|html|dataset|on<Event>|attr. kids: nodes
// or strings (text), nested arrays flattened, null/false skipped.
function el(tag, props, ...kids) {
  const n = document.createElement(tag);
  if (props) {
    for (const [k, v] of Object.entries(props)) {
      if (v == null) continue;
      if (k === "class") n.className = v;
      else if (k === "html") n.innerHTML = v;
      else if (k === "dataset") Object.assign(n.dataset, v);
      else if (k.startsWith("on") && typeof v === "function") n[k.toLowerCase()] = v;
      else n.setAttribute(k, v);
    }
  }
  for (const kid of kids.flat()) {
    if (kid == null || kid === false) continue;
    n.appendChild(typeof kid === "string" ? document.createTextNode(kid) : kid);
  }
  return n;
}

// Lucide-style stroke icons, inner markup only.
const ICONS = {
  home: '<path d="M3 9.5 12 3l9 6.5"/><path d="M5 10v10h6v-6h2v6h6V10"/>',
  shield: '<path d="M12 3l8 3v6c0 5-3.5 7.6-8 9-4.5-1.4-8-4-8-9V6z"/>',
  coins: '<circle cx="8" cy="8" r="5"/><path d="M18.1 10.4a5 5 0 1 1-6.7 6.7"/>',
  trophy: '<path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0z"/><path d="M5 5H3v2a3 3 0 0 0 3 3M19 5h2v2a3 3 0 0 1-3 3"/>',
  gift: '<path d="M20 12v9H4v-9M2 7h20v5H2zM12 22V7M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7zM12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7z"/>',
  ticket: '<path d="M3 9a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2 2 2 0 0 0 0 6 2 2 0 0 1-2 2H5a2 2 0 0 1-2-2 2 2 0 0 0 0-6z"/><path d="M13 7v10"/>',
  ban: '<circle cx="12" cy="12" r="9"/><path d="m5.6 5.6 12.8 12.8"/>',
  sparkles: '<path d="M12 3l1.8 4.7L18.5 9.5l-4.7 1.8L12 16l-1.8-4.7L5.5 9.5l4.7-1.8z"/><path d="M19 14l.8 2.2L22 17l-2.2.8L19 20l-.8-2.2L16 17l2.2-.8z"/>',
  scroll: '<path d="M8 3h11a2 2 0 0 1 2 2v3H8zM3 8h13v11a2 2 0 0 1-2 2H6a3 3 0 0 1-3-3z"/>',
  bot: '<rect x="4" y="8" width="16" height="12" rx="2"/><path d="M12 8V4M9 14h.01M15 14h.01"/>',
  bell: '<path d="M6 8a6 6 0 0 1 12 0c0 7 3 8 3 8H3s3-1 3-8M10.3 21a1.9 1.9 0 0 0 3.4 0"/>',
  badge: '<path d="M12 2l2.4 1.8 3-.2 1 2.8 2.4 1.6-.9 2.9.9 2.9-2.4 1.6-1 2.8-3-.2L12 22l-2.4-1.8-3 .2-1-2.8L3.2 16l.9-2.9L3.2 10l2.4-1.6 1-2.8 3 .2z"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-2.9 1.2V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-2.9-1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1A1.7 1.7 0 0 0 2.6 15a2 2 0 1 1 0-4 1.7 1.7 0 0 0 1.5-2.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1A1.7 1.7 0 0 0 9 3.6 2 2 0 1 1 13 3.6a1.7 1.7 0 0 0 2.9 1.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1A1.7 1.7 0 0 0 21.4 11a2 2 0 1 1 0 4z"/>',
  logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/>',
  chevron: '<path d="m6 9 6 6 6-6"/>',
  users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M22 21v-2a4 4 0 0 0-3-3.9M16 3.1A4 4 0 0 1 16 11"/>',
  hash: '<path d="M4 9h16M4 15h16M10 3 8 21M16 3l-2 18"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  rocket: '<path d="M5 13c-1.5.6-2.5 2-3 4 2 0 3.4-.9 4-3M9 13l-2 2M11 15l2-2M15 7a2 2 0 1 1-3 3M14 16s4-1.5 5.5-4S20 3 20 3s-3.5-.5-6 1-4 5-4 5l3 3z"/>',
};

function icon(name, cls) {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-width", "2");
  svg.setAttribute("stroke-linecap", "round");
  svg.setAttribute("stroke-linejoin", "round");
  if (cls) svg.setAttribute("class", cls);
  svg.innerHTML = ICONS[name] || ICONS.settings;
  return svg;
}

// Module display metadata; unknown modules fall back to a titled gear.
const MODMETA = {
  moderation: { label: "Moderation", icon: "shield" },
  automod: { label: "AutoMod", icon: "ban" },
  economy: { label: "Economy", icon: "coins" },
  leveling: { label: "Leveling", icon: "trophy" },
  tickets: { label: "Tickets", icon: "ticket" },
  giveaways: { label: "Giveaways", icon: "gift" },
  logging: { label: "Logging", icon: "scroll" },
  verification: { label: "Verification", icon: "badge" },
  welcome: { label: "Welcome", icon: "bell" },
  utility: { label: "Utility", icon: "settings" },
  ai: { label: "AI", icon: "bot" },
};

function modMeta(name) {
  return MODMETA[name] || { label: titleCase(name), icon: "settings" };
}

function titleCase(s) {
  return String(s || "").replace(/(^|[\s_-])\w/g, (m) => m.toUpperCase()).replace(/[_-]/g, " ");
}

/* --------------------------------------------------------------- api/toast */

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

let toastTimer;
function toast(msg, isErr) {
  const t = $("#toast");
  $(".msg", t).textContent = msg;
  t.classList.toggle("err", !!isErr);
  t.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove("show"), 2600);
}

/* --------------------------------------------------------------- app state */

const state = {
  me: null,
  guildID: null,
  roles: [], // [{id,name,color,position,managed,hoist,everyone}]
  channels: [], // [{id,name,type,parent_id,position}]
  cacheKey: null, // guild whose roles/channels are loaded
};

const TEXT_CHANNEL_TYPES = new Set([0, 5, 15]); // text, announcement, forum
const CATEGORY_TYPE = 4;

function assignableRoles() {
  return state.roles.filter((r) => !r.everyone && !r.managed);
}

function roleByID(id) {
  return state.roles.find((r) => r.id === id);
}

// textChannelGroups returns [{category, channels:[…]}] for the channel picker,
// channels without a parent collected under a null category.
function textChannelGroups() {
  const cats = new Map(); // parentID -> {name, channels}
  for (const c of state.channels) {
    if (c.type === CATEGORY_TYPE) {
      if (!cats.has(c.id)) cats.set(c.id, { name: c.name, channels: [] });
      else cats.get(c.id).name = c.name;
    }
  }
  const loose = [];
  for (const c of state.channels) {
    if (!TEXT_CHANNEL_TYPES.has(c.type)) continue;
    if (c.parent_id && cats.has(c.parent_id)) cats.get(c.parent_id).channels.push(c);
    else loose.push(c);
  }
  const groups = [];
  if (loose.length) groups.push({ category: null, channels: loose });
  for (const { name, channels } of cats.values()) {
    if (channels.length) groups.push({ category: name, channels });
  }
  return groups;
}

/* ----------------------------------------------------------- field widgets */

// fieldRow builds a labelled config control. Returns {node, key, read()}.
function fieldRow(field, value) {
  const meta = el("div", { class: "meta" },
    el("div", { class: "l" }, field.label),
    field.help ? el("div", { class: "help" }, field.help) : null);

  if (field.type === "bool") {
    const input = el("input", { type: "checkbox" });
    input.checked = !!value;
    const sw = el("label", { class: "switch" }, input, el("span", { class: "track" }));
    return {
      node: el("div", { class: "field field-row" }, meta, sw),
      key: field.key,
      read: () => input.checked,
    };
  }

  if (field.type === "int") {
    const input = el("input", { class: "input", type: "number", style: "max-width:160px" });
    if (field.min || field.max) {
      if (field.min) input.min = field.min;
      if (field.max) input.max = field.max;
    }
    input.value = value ?? 0;
    return {
      node: el("div", { class: "field field-row" }, meta, input),
      key: field.key,
      read: () => Number(input.value),
    };
  }

  if (field.type === "channel" || field.type === "role") {
    const select = el("select", { class: "input", style: "max-width:280px" });
    select.appendChild(el("option", { value: "" }, "— None —"));

    if (field.type === "role") {
      for (const r of assignableRoles()) {
        select.appendChild(el("option", { value: r.id }, r.name));
      }
    } else {
      for (const g of textChannelGroups()) {
        const parent = g.category ? el("optgroup", { label: g.category }) : select;
        for (const c of g.channels) {
          parent.appendChild(el("option", { value: c.id }, "# " + c.name));
        }
        if (g.category) select.appendChild(parent);
      }
    }
    const cur = value ?? "";
    // Keep an unknown/stale id selectable so saving doesn't silently drop it.
    if (cur && ![...select.options].some((o) => o.value === cur)) {
      select.appendChild(el("option", { value: cur }, `Unknown (${cur})`));
    }
    select.value = cur;

    let control = select;
    if (field.type === "role") {
      const sw = el("span", { class: "swatch" });
      const paint = () => {
        const r = roleByID(select.value);
        sw.style.background = r && r.color ? r.color : "var(--border-2)";
      };
      paint();
      select.onchange = paint;
      control = el("div", { class: "role-field" }, sw, select);
    }
    return {
      node: el("div", { class: "field field-row" }, meta, control),
      key: field.key,
      read: () => select.value.trim(),
    };
  }

  // string (and any unknown type) → text input
  const input = el("input", { class: "input", type: "text", style: "max-width:280px" });
  input.value = value ?? "";
  if (field.maxlen) input.maxLength = field.maxlen;
  return {
    node: el("div", { class: "field field-row" }, meta, input),
    key: field.key,
    read: () => input.value.trim(),
  };
}

/* ------------------------------------------------------------------- pages */

function pageHead(title, sub, ic) {
  return el("div", { class: "page-head" },
    ic ? el("div", { class: "gicon" }, icon(ic)) : null,
    el("div", {},
      el("h1", {}, title),
      sub ? el("div", { class: "sub" }, sub) : null));
}

function spinner() {
  return el("div", { class: "empty" }, el("div", { class: "spinner" }));
}

function emptyState(ic, title, hint) {
  return el("div", { class: "empty" }, icon(ic),
    el("div", { class: "t" }, title), hint ? el("div", {}, hint) : null);
}

// loadGuildData fetches roles + channels once per guild and caches them so the
// pickers don't refetch on every navigation.
async function loadGuildData(id) {
  if (state.cacheKey === id) return;
  const [roles, channels] = await Promise.all([
    api("GET", `/api/guilds/${id}/roles`),
    api("GET", `/api/guilds/${id}/channels`),
  ]);
  state.roles = roles || [];
  state.channels = channels || [];
  state.cacheKey = id;
}

const fmtNum = (n) => (n ?? 0).toLocaleString();

async function pageOverview(root) {
  const id = state.guildID;
  root.appendChild(spinner());
  let ov, audit;
  try {
    [ov, audit] = await Promise.all([
      api("GET", `/api/guilds/${id}/overview`),
      api("GET", `/api/guilds/${id}/audit`).catch(() => []),
    ]);
  } catch (e) {
    root.replaceChildren(emptyState("shield", "Couldn't load server", e.message));
    return;
  }

  const stats = [
    ["users", "Members", fmtNum(ov.members)],
    ["hash", "Channels", fmtNum(ov.channels)],
    ["shield", "Roles", fmtNum(ov.roles)],
    ["rocket", "Boost tier", `${ov.premium_tier} <small>· ${fmtNum(ov.boosts)} boosts</small>`],
  ];
  const statGrid = el("div", { class: "stats" },
    ...stats.map(([ic, k, v]) =>
      el("div", { class: "stat" },
        el("div", { class: "k" }, icon(ic), k),
        el("div", { class: "v", html: v }))));

  const created = new Date(ov.created_at);
  const metaCard = el("div", { class: "card" },
    el("div", { class: "card-head" }, el("h2", {}, "Server")),
    el("div", { class: "fields" },
      kv("Server ID", el("span", { class: "chip" }, ov.id)),
      kv("Owner ID", el("span", { class: "chip" }, ov.owner_id)),
      kv("Created", el("span", {}, isNaN(created) ? "—" : created.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" })))));

  root.replaceChildren(
    pageHead(ov.name, "Server overview", null),
    statGrid,
    metaCard,
    recentChanges(audit));
}

function kv(label, valueNode) {
  return el("div", { class: "field field-row" },
    el("div", { class: "meta" }, el("div", { class: "l" }, label)), valueNode);
}

function recentChanges(rows) {
  const card = el("div", { class: "card" },
    el("div", { class: "card-head" },
      el("h2", {}, "Recent changes"),
      el("div", { class: "sub" }, "Latest configuration edits from the dashboard")));
  if (!rows || !rows.length) {
    card.appendChild(emptyState("clock", "No changes yet", "Edits you make here will show up in this list."));
    return card;
  }
  const body = el("tbody");
  for (const r of rows) {
    const when = new Date(r.created_at);
    const fields = Object.keys(r.changes || {});
    body.appendChild(el("tr", {},
      el("td", {}, modMeta(r.module).label),
      el("td", {}, el("span", { class: "muted" }, r.username || r.user_id)),
      el("td", {}, ...(fields.length
        ? fields.map((k) => el("span", { class: "chip", style: "margin:0 4px 4px 0" }, k))
        : [el("span", { class: "muted" }, "—")])),
      el("td", { class: "muted" }, isNaN(when) ? "—" : when.toLocaleString())));
  }
  card.appendChild(el("div", { class: "table-wrap" },
    el("table", { class: "table" },
      el("thead", {}, el("tr", {},
        el("th", {}, "Module"), el("th", {}, "By"),
        el("th", {}, "Fields"), el("th", {}, "When"))),
      body)));
  return card;
}

async function pageModule(root, name) {
  const id = state.guildID;
  root.appendChild(spinner());
  let mod;
  try {
    await loadGuildData(id);
    mod = await api("GET", `/api/guilds/${id}/modules/${name}`);
  } catch (e) {
    root.replaceChildren(emptyState("settings", "Couldn't load module", e.message));
    return;
  }
  const meta = modMeta(mod.module);

  const card = el("div", { class: "card" });
  if (!mod.fields || !mod.fields.length) {
    card.appendChild(emptyState("settings", "Nothing to configure", "This module has no settings yet."));
  } else {
    const rows = mod.fields.map((f) => fieldRow(f, (mod.values || {})[f.key]));
    card.appendChild(el("div", { class: "fields" }, ...rows.map((r) => r.node)));

    const save = el("button", { class: "btn btn-primary" }, "Save changes");
    save.onclick = async () => {
      const patch = {};
      for (const r of rows) patch[r.key] = r.read();
      save.disabled = true;
      try {
        const updated = await api("PATCH", `/api/guilds/${id}/modules/${mod.module}`, patch);
        if (updated && updated.values) mod.values = updated.values;
        toast(`${meta.label} saved`);
      } catch (e) {
        toast(e.message, true);
      } finally {
        save.disabled = false;
      }
    };
    card.appendChild(el("div", { class: "card-actions" }, save));
  }

  root.replaceChildren(
    pageHead(meta.label, `Configure the ${meta.label.toLowerCase()} module`, meta.icon),
    card);
}

async function pageAudit(root) {
  const id = state.guildID;
  root.appendChild(spinner());
  let rows;
  try {
    rows = await api("GET", `/api/guilds/${id}/audit`);
  } catch (e) {
    root.replaceChildren(emptyState("scroll", "Couldn't load audit log", e.message));
    return;
  }
  root.replaceChildren(
    pageHead("Audit log", "Every configuration change made from the dashboard", null),
    recentChanges(rows));
}

/* ------------------------------------------------------------------ router */

const ROUTES = {
  overview: (root) => pageOverview(root),
  audit: (root) => pageAudit(root),
};

function currentRoute() {
  const h = (location.hash || "#/overview").replace(/^#\/?/, "");
  const parts = h.split("/").filter(Boolean);
  if (parts[0] === "m" && parts[1]) return { kind: "module", name: parts[1] };
  if (parts[0] === "audit") return { kind: "audit" };
  return { kind: "overview" };
}

async function router() {
  if (!state.guildID) return;
  const route = currentRoute();
  const root = $("#content");
  root.replaceChildren();
  markActiveNav(route);
  if (route.kind === "module") return pageModule(root, route.name);
  if (route.kind === "audit") return pageAudit(root);
  return pageOverview(root);
}

function markActiveNav(route) {
  for (const a of document.querySelectorAll("#nav a")) {
    const r = a.dataset.route;
    const active =
      (route.kind === "overview" && r === "overview") ||
      (route.kind === "audit" && r === "audit") ||
      (route.kind === "module" && r === `m/${route.name}`);
    a.classList.toggle("active", active);
  }
}

/* ------------------------------------------------------------------- shell */

function navLink(route, label, ic) {
  return el("a", { dataset: { route }, href: `#/${route}` }, icon(ic), el("span", {}, label));
}

async function buildNav() {
  const nav = $("#nav");
  nav.replaceChildren(navLink("overview", "Overview", "home"));

  let mods = [];
  try {
    mods = await api("GET", `/api/guilds/${state.guildID}/modules`);
  } catch {
    /* nav still usable without the module list */
  }
  if (mods.length) {
    nav.appendChild(el("div", { class: "nav-label" }, "Modules"));
    for (const m of mods) {
      const meta = modMeta(m.module);
      nav.appendChild(navLink(`m/${m.module}`, meta.label, meta.icon));
    }
  }
  nav.appendChild(el("div", { class: "nav-label" }, "Logs"));
  nav.appendChild(navLink("audit", "Audit log", "scroll"));
}

function guildIcon(g, cls) {
  const wrap = el("div", { class: "gicon" + (cls ? " " + cls : "") });
  if (g && g.icon) {
    wrap.appendChild(el("img", { src: `https://cdn.discordapp.com/icons/${g.id}/${g.icon}.png?size=64`, alt: "" }));
  } else {
    wrap.textContent = (g && g.name ? g.name[0] : "?").toUpperCase();
  }
  return wrap;
}

// closeMenus removes any open dropdown and its outside-click listener.
let activeMenu = null;
function closeMenus() {
  if (activeMenu) { activeMenu.remove(); activeMenu = null; }
  document.removeEventListener("click", onDocClick, true);
}
function onDocClick(e) {
  if (activeMenu && !activeMenu.contains(e.target)) closeMenus();
}
function openMenu(menu, anchor) {
  closeMenus();
  anchor.appendChild(menu);
  activeMenu = menu;
  // Defer so the click that opened it doesn't immediately close it.
  setTimeout(() => document.addEventListener("click", onDocClick, true), 0);
}

function renderPicker() {
  const slot = $("#picker-slot");
  const guild = state.me.guilds.find((g) => g.id === state.guildID);
  const wrap = el("div", { class: "picker" });
  const btn = el("button", { class: "picker-btn" },
    guildIcon(guild), el("span", {}, guild ? guild.name : "Select server"), icon("chevron"));
  btn.onclick = (e) => {
    e.stopPropagation();
    if (activeMenu) return closeMenus();
    const menu = el("div", { class: "picker-menu" });
    for (const g of state.me.guilds) {
      menu.appendChild(el("button", { onClick: () => { closeMenus(); selectGuild(g.id); } },
        guildIcon(g), el("span", {}, g.name)));
    }
    openMenu(menu, wrap);
  };
  wrap.appendChild(btn);
  slot.replaceChildren(wrap);
}

function renderUser() {
  const slot = $("#user-slot");
  const me = state.me;
  const wrap = el("div", { class: "picker" });
  const avatar = el("div", { class: "gicon" });
  if (me.avatar) avatar.appendChild(el("img", { src: `https://cdn.discordapp.com/avatars/${me.user_id}/${me.avatar}.png?size=64`, alt: "" }));
  else avatar.textContent = (me.username || "?")[0].toUpperCase();

  const btn = el("button", { class: "picker-btn" }, avatar, el("span", {}, me.username), icon("chevron"));
  btn.onclick = (e) => {
    e.stopPropagation();
    if (activeMenu) return closeMenus();
    const menu = el("div", { class: "picker-menu", style: "left:auto;right:0" });
    const out = el("button", { onClick: async () => { await api("POST", "/auth/logout"); location.reload(); } },
      icon("logout"), el("span", {}, "Log out"));
    menu.appendChild(out);
    openMenu(menu, wrap);
  };
  wrap.appendChild(btn);
  slot.replaceChildren(wrap);
}

function renderFooter() {
  const foot = $("#sb-foot");
  const guild = state.me.guilds.find((g) => g.id === state.guildID);
  if (!guild) { foot.replaceChildren(); return; }
  foot.replaceChildren(guildIcon(guild), el("span", { class: "name" }, guild.name));
}

async function selectGuild(id) {
  state.guildID = id;
  state.cacheKey = null;
  renderPicker();
  renderFooter();
  await buildNav();
  await router();
}

/* -------------------------------------------------------------------- init */

async function init() {
  let me;
  try {
    me = await api("GET", "/api/me");
  } catch {
    $("#login").classList.remove("hidden");
    $("#app").classList.add("hidden");
    return;
  }
  state.me = me;
  $("#login").classList.add("hidden");
  $("#app").classList.remove("hidden");
  renderUser();

  if (!me.guilds || !me.guilds.length) {
    renderPicker();
    $("#content").replaceChildren(
      emptyState("shield", "No manageable servers",
        "You need the Manage Server permission on a server that has disgo."));
    return;
  }

  window.addEventListener("hashchange", router);
  await selectGuild(me.guilds[0].id);
}

init();
