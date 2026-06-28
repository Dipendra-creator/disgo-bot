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
  features: {}, // management consoles available for the current guild
  mod: { target: "", action: "", offset: 0 }, // moderation console filter/pager state
  econ: { offset: 0 }, // economy leaderboard pager state
  lvl: { offset: 0 }, // leveling leaderboard pager state
  tk: { status: "active", offset: 0 }, // tickets console filter/pager state
  gw: { offset: 0 }, // giveaways pager state
};

const TEXT_CHANNEL_TYPES = new Set([0, 5, 15]); // text, announcement, forum
const CATEGORY_TYPE = 4;

function assignableRoles() {
  return state.roles.filter((r) => !r.everyone && !r.managed);
}

function roleByID(id) {
  return state.roles.find((r) => r.id === id);
}

// roleChip renders a role as a coloured swatch + name, or a muted dash when the
// id is empty or the role is no longer in the guild.
function roleChip(id) {
  if (!id) return el("span", { class: "muted" }, "—");
  const r = roleByID(id);
  if (!r) return el("span", { class: "chip" }, id);
  return el("span", { class: "role-chip" },
    el("span", { class: "swatch", style: `background:${r.color || "#3f4147"}` }),
    el("span", {}, r.name));
}

// roleSelect builds a native <select> of assignable roles with a "— None —"
// option, preselecting currentID (kept even if it's now stale).
function roleSelect(currentID, withNone) {
  const sel = el("select", { class: "input" });
  if (withNone !== false) sel.appendChild(el("option", { value: "" }, "— None —"));
  let matched = false;
  for (const r of assignableRoles()) {
    const opt = el("option", { value: r.id }, r.name);
    if (r.id === currentID) { opt.selected = true; matched = true; }
    sel.appendChild(opt);
  }
  if (currentID && !matched) {
    const opt = el("option", { value: currentID, selected: true }, `Unknown role (${currentID})`);
    sel.appendChild(opt);
  }
  return sel;
}

// channelSelect builds a native <select> of text channels grouped by category,
// preselecting currentID (kept even if it's now stale). No "none" option — a
// channel is required by its callers.
function channelSelect(currentID) {
  const sel = el("select", { class: "input" });
  sel.appendChild(el("option", { value: "" }, "— Pick a channel —"));
  let matched = false;
  for (const g of textChannelGroups()) {
    const parent = g.category ? el("optgroup", { label: g.category }) : sel;
    for (const c of g.channels) {
      const opt = el("option", { value: c.id }, "# " + c.name);
      if (c.id === currentID) { opt.selected = true; matched = true; }
      parent.appendChild(opt);
    }
    if (g.category) sel.appendChild(parent);
  }
  if (currentID && !matched) {
    sel.appendChild(el("option", { value: currentID, selected: true }, `Unknown (${currentID})`));
  }
  return sel;
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

/* ------------------------------------------------ moderation console (Inc2) */

const ACTION_LABEL = {
  ban: "Ban", unban: "Unban", kick: "Kick",
  timeout: "Timeout", untimeout: "Untimeout", warn: "Warn",
};

function actionBadge(a) {
  return el("span", { class: "badge act-" + a }, ACTION_LABEL[a] || a);
}

// confirmDialog shows a modal and resolves true/false. Used to gate destructive
// actions (ban/kick) behind an explicit confirmation.
function confirmDialog(title, message, danger) {
  return new Promise((resolve) => {
    const close = (v) => { overlay.remove(); resolve(v); };
    const overlay = el("div", { class: "modal-overlay" },
      el("div", { class: "modal" },
        el("h3", {}, title),
        el("p", { class: "muted" }, message),
        el("div", { class: "modal-actions" },
          el("button", { class: "btn", onClick: () => close(false) }, "Cancel"),
          el("button", { class: "btn " + (danger ? "btn-danger" : "btn-primary"), onClick: () => close(true) }, "Confirm"))));
    overlay.onclick = (e) => { if (e.target === overlay) close(false); };
    document.body.appendChild(overlay);
  });
}

function actionFormCard(id) {
  const action = el("select", { class: "input", style: "max-width:150px" },
    el("option", { value: "warn" }, "Warn"),
    el("option", { value: "timeout" }, "Timeout"),
    el("option", { value: "kick" }, "Kick"),
    el("option", { value: "ban" }, "Ban"));
  const target = el("input", { class: "input", type: "text", placeholder: "User ID", style: "max-width:190px" });
  const dur = el("input", { class: "input", type: "number", min: "1", value: "10", style: "max-width:90px" });
  const durRow = el("label", { class: "dur hidden" }, el("span", { class: "muted" }, "Minutes"), dur);
  const reason = el("input", { class: "input", type: "text", placeholder: "Reason (optional)", style: "flex:1;min-width:180px" });
  action.onchange = () => durRow.classList.toggle("hidden", action.value !== "timeout");

  const submit = el("button", { class: "btn btn-primary" }, "Apply");
  submit.onclick = async () => {
    const act = action.value;
    const tid = target.value.trim();
    if (!tid) { toast("Target user ID is required", true); return; }
    if (act === "ban" || act === "kick") {
      const ok = await confirmDialog(
        `${ACTION_LABEL[act]} user?`,
        `This ${act}s ${tid} on Discord immediately and records a case.`, true);
      if (!ok) return;
    }
    const body = { action: act, target_id: tid, reason: reason.value.trim() };
    if (act === "timeout") body.duration_ms = Math.max(1, Number(dur.value)) * 60000;
    submit.disabled = true;
    try {
      const c = await api("POST", `/api/guilds/${id}/moderation/actions`, body);
      toast(`Case #${c.number} · ${ACTION_LABEL[c.action] || c.action}`);
      target.value = ""; reason.value = "";
      state.mod.offset = 0;
      router();
    } catch (e) {
      toast(e.message, true);
    } finally {
      submit.disabled = false;
    }
  };

  return el("div", { class: "card" },
    el("div", { class: "card-head" },
      el("h2", {}, "Take action"),
      el("div", { class: "sub" }, "Applies immediately and is recorded as a case.")),
    el("div", { class: "toolbar", style: "margin-top:16px" }, action, target, durRow, reason, submit));
}

async function editReason(id, c, cell, btn) {
  const input = el("input", { class: "input", type: "text", style: "min-width:200px" });
  input.value = c.reason || "";
  const restore = () => {
    cell.replaceChildren(c.reason || el("span", { class: "muted" }, "—"));
    btn.disabled = false;
  };
  const save = el("button", { class: "btn btn-sm btn-primary" }, "Save");
  const cancel = el("button", { class: "btn btn-sm btn-ghost" }, "Cancel");
  cancel.onclick = restore;
  save.onclick = async () => {
    const v = input.value.trim();
    if (!v) { toast("Reason can't be empty", true); return; }
    save.disabled = true;
    try {
      const updated = await api("PATCH", `/api/guilds/${id}/moderation/cases/${c.number}`, { reason: v });
      c.reason = updated.reason;
      toast(`Case #${c.number} updated`);
      restore();
    } catch (e) {
      toast(e.message, true);
      save.disabled = false;
    }
  };
  btn.disabled = true;
  cell.replaceChildren(el("div", { class: "edit-row" }, input, save, cancel));
  input.focus();
}

function caseRow(id, c) {
  const when = new Date(c.created_at);
  const reasonCell = el("td", {}, c.reason || el("span", { class: "muted" }, "—"));
  const edit = el("button", { class: "btn btn-ghost btn-sm", title: "Edit reason" }, icon("settings"));
  edit.onclick = () => editReason(id, c, reasonCell, edit);
  return el("tr", {},
    el("td", {}, el("span", { class: "chip" }, "#" + c.number)),
    el("td", {}, actionBadge(c.action)),
    el("td", {}, el("span", { class: "chip" }, c.target_id)),
    el("td", {}, c.moderator_id ? el("span", { class: "chip" }, c.moderator_id) : el("span", { class: "muted" }, "system")),
    reasonCell,
    el("td", { class: "muted" }, isNaN(when) ? "—" : when.toLocaleDateString()),
    el("td", { style: "text-align:right" }, edit));
}

async function renderCases(card, id, limit) {
  card.replaceChildren(spinner());

  const fTarget = el("input", { class: "input", type: "text", placeholder: "Filter by user ID", style: "max-width:190px" });
  fTarget.value = state.mod.target;
  const fAction = el("select", { class: "input", style: "max-width:150px" },
    el("option", { value: "" }, "All actions"),
    ...Object.keys(ACTION_LABEL).map((a) => el("option", { value: a }, ACTION_LABEL[a])));
  fAction.value = state.mod.action;
  const apply = el("button", { class: "btn" }, "Filter");
  apply.onclick = () => {
    state.mod.target = fTarget.value.trim();
    state.mod.action = fAction.value;
    state.mod.offset = 0;
    renderCases(card, id, limit);
  };
  fTarget.onkeydown = (e) => { if (e.key === "Enter") apply.onclick(); };

  let page;
  try {
    const qs = new URLSearchParams({ limit: String(limit), offset: String(state.mod.offset) });
    if (state.mod.target) qs.set("target", state.mod.target);
    if (state.mod.action) qs.set("action", state.mod.action);
    page = await api("GET", `/api/guilds/${id}/moderation/cases?${qs}`);
  } catch (e) {
    card.replaceChildren(emptyState("shield", "Couldn't load cases", e.message));
    return;
  }

  card.replaceChildren(
    el("div", { class: "card-head" },
      el("h2", {}, "Cases"),
      el("div", { class: "sub" }, `${page.total} total`)),
    el("div", { class: "toolbar", style: "margin-top:16px" }, fTarget, fAction, apply));

  if (!page.cases.length) {
    card.appendChild(emptyState("scroll", "No cases", "Nothing matches these filters yet."));
    return;
  }

  const body = el("tbody");
  for (const c of page.cases) body.appendChild(caseRow(id, c));
  card.appendChild(el("div", { class: "table-wrap" },
    el("table", { class: "table" },
      el("thead", {}, el("tr", {},
        el("th", {}, "#"), el("th", {}, "Action"), el("th", {}, "Target"),
        el("th", {}, "Moderator"), el("th", {}, "Reason"), el("th", {}, "When"), el("th", {}, ""))),
      body)));

  const start = state.mod.offset;
  const end = state.mod.offset + page.cases.length;
  const prev = el("button", { class: "btn btn-sm" }, "Prev");
  prev.disabled = start === 0;
  prev.onclick = () => { state.mod.offset = Math.max(0, start - limit); renderCases(card, id, limit); };
  const next = el("button", { class: "btn btn-sm" }, "Next");
  next.disabled = end >= page.total;
  next.onclick = () => { state.mod.offset = start + limit; renderCases(card, id, limit); };
  card.appendChild(el("div", { class: "pager" },
    el("span", { class: "muted" }, `${start + 1}–${end} of ${page.total}`), prev, next));
}

async function pageModeration(root) {
  const id = state.guildID;
  root.replaceChildren(
    pageHead("Moderation", "Browse cases and take manual action", "shield"),
    actionFormCard(id));
  const listCard = el("div", { class: "card" });
  root.appendChild(listCard);
  await renderCases(listCard, id, 25);
}

/* -------------------------------------------------- economy console (Inc3) */

// pager renders a Prev/Next footer for an offset-based list. onGo(newOffset)
// re-renders the caller.
function pager(offset, shown, total, limit, onGo) {
  const start = offset;
  const end = offset + shown;
  const prev = el("button", { class: "btn btn-sm" }, "Prev");
  prev.disabled = start === 0;
  prev.onclick = () => onGo(Math.max(0, start - limit));
  const next = el("button", { class: "btn btn-sm" }, "Next");
  next.disabled = end >= total;
  next.onclick = () => onGo(start + limit);
  return el("div", { class: "pager" },
    el("span", { class: "muted" }, total ? `${start + 1}–${end} of ${total}` : "0 of 0"), prev, next);
}

// memberCell renders a leaderboard member as name (when known) over a muted id.
function memberCell(m) {
  if (m.username) {
    return el("td", {},
      el("div", {}, m.username),
      el("div", { class: "muted mono", style: "font-size:11px" }, m.user_id));
  }
  return el("td", {}, el("span", { class: "chip" }, m.user_id));
}

// userCell renders a name (when known) over a muted id, or a bare id chip. The
// caller wraps it in a <td>.
function userCell(name, id) {
  if (name) {
    return el("div", {},
      el("div", {}, name),
      el("div", { class: "muted mono", style: "font-size:11px" }, id));
  }
  return el("span", { class: "chip" }, id);
}

// channelName maps a channel id to its "# name" from the cache, or the raw id.
function channelName(id) {
  const c = state.channels.find((x) => x.id === id);
  return c ? "# " + c.name : id;
}

const ECON_LIMIT = 25;

async function renderEconLeaderboard(card) {
  card.replaceChildren(spinner());
  let page;
  try {
    const qs = new URLSearchParams({ limit: String(ECON_LIMIT), offset: String(state.econ.offset) });
    page = await api("GET", `/api/guilds/${state.guildID}/economy/leaderboard?${qs}`);
  } catch (e) {
    card.replaceChildren(emptyState("coins", "Couldn't load leaderboard", e.message));
    return;
  }
  const sym = page.symbol || "";
  card.replaceChildren(el("div", { class: "card-head" },
    el("h2", {}, "Net worth"),
    el("div", { class: "sub" }, `${page.total} ranked · ${page.currency || "coins"}`)));

  if (!page.members.length) {
    card.appendChild(emptyState("coins", "No balances yet", "Members appear here once they earn currency."));
    return;
  }
  const body = el("tbody");
  page.members.forEach((m, i) => {
    body.appendChild(el("tr", {},
      el("td", { class: "muted" }, "#" + (state.econ.offset + i + 1)),
      memberCell(m),
      el("td", { style: "text-align:right" }, `${sym}${fmtNum(m.wallet)}`),
      el("td", { style: "text-align:right" }, `${sym}${fmtNum(m.bank)}`),
      el("td", { style: "text-align:right;font-weight:600" }, `${sym}${fmtNum(m.net)}`)));
  });
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {},
      el("th", {}, "Rank"), el("th", {}, "Member"),
      el("th", { style: "text-align:right" }, "Wallet"),
      el("th", { style: "text-align:right" }, "Bank"),
      el("th", { style: "text-align:right" }, "Net"))),
    body)));
  card.appendChild(pager(state.econ.offset, page.members.length, page.total, ECON_LIMIT,
    (o) => { state.econ.offset = o; renderEconLeaderboard(card); }));
}

// shopItemForm builds the add/edit fields for a shop item. Returns
// {node, read()} where read() yields the request body.
function shopItemForm(item) {
  const it = item || { name: "", description: "", price: 0, role_id: "", stock: -1 };
  const name = el("input", { class: "input", type: "text", placeholder: "Name", value: it.name, style: "max-width:200px" });
  const desc = el("input", { class: "input", type: "text", placeholder: "Description", value: it.description, style: "flex:1;min-width:180px" });
  const price = el("input", { class: "input", type: "number", min: "0", value: String(it.price), style: "max-width:120px" });
  const role = roleSelect(it.role_id, true);
  role.style.maxWidth = "180px";

  const unlimited = it.stock < 0;
  const stock = el("input", { class: "input", type: "number", min: "0", value: String(unlimited ? 0 : it.stock), style: "max-width:90px" });
  stock.disabled = unlimited;
  const unl = el("input", { type: "checkbox" });
  unl.checked = unlimited;
  unl.onchange = () => { stock.disabled = unl.checked; };
  const unlLabel = el("label", { class: "dur" }, unl, el("span", { class: "muted" }, "Unlimited"));

  const node = el("div", { class: "toolbar" }, name, desc, price,
    el("label", { class: "dur" }, el("span", { class: "muted" }, "Role"), role),
    el("label", { class: "dur" }, el("span", { class: "muted" }, "Stock"), stock), unlLabel);
  return {
    node,
    read: () => ({
      name: name.value.trim(),
      description: desc.value.trim(),
      price: Math.max(0, Number(price.value) || 0),
      role_id: role.value,
      stock: unl.checked ? -1 : Math.max(0, Number(stock.value) || 0),
    }),
  };
}

async function renderShop(card) {
  card.replaceChildren(spinner());
  let page;
  try {
    page = await api("GET", `/api/guilds/${state.guildID}/economy/shop?limit=100`);
  } catch (e) {
    card.replaceChildren(emptyState("coins", "Couldn't load shop", e.message));
    return;
  }

  const form = shopItemForm(null);
  const addBtn = el("button", { class: "btn btn-primary" }, "Add item");
  addBtn.onclick = async () => {
    const body = form.read();
    if (!body.name) { toast("Item name is required", true); return; }
    addBtn.disabled = true;
    try {
      await api("POST", `/api/guilds/${state.guildID}/economy/shop`, body);
      toast(`Added “${body.name}”`);
      renderShop(card);
    } catch (e) {
      toast(e.message, true);
      addBtn.disabled = false;
    }
  };

  card.replaceChildren(
    el("div", { class: "card-head" }, el("h2", {}, "Shop"),
      el("div", { class: "sub" }, `${page.total} item${page.total === 1 ? "" : "s"}`)),
    el("div", { class: "stack", style: "margin-top:16px" }, form.node,
      el("div", { class: "card-actions", style: "margin-top:0;border:0;padding:0" }, addBtn)));

  if (!page.items.length) {
    card.appendChild(emptyState("coins", "No items", "Add a shop item above to get started."));
    return;
  }
  const body = el("tbody");
  for (const it of page.items) body.appendChild(shopRow(card, it));
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {},
      el("th", {}, "Item"), el("th", {}, "Description"),
      el("th", { style: "text-align:right" }, "Price"), el("th", {}, "Role"),
      el("th", {}, "Stock"), el("th", {}, ""))),
    body)));
}

function shopRow(card, it) {
  const edit = el("button", { class: "btn btn-ghost btn-sm", title: "Edit" }, icon("settings"));
  edit.onclick = () => editShopItem(card, it);
  const del = el("button", { class: "btn btn-ghost btn-sm btn-danger", title: "Delete" }, icon("ban"));
  del.onclick = async () => {
    if (!(await confirmDialog("Delete item?", `Remove “${it.name}” from the shop.`, true))) return;
    try {
      await api("DELETE", `/api/guilds/${state.guildID}/economy/shop/${it.id}`);
      toast(`Deleted “${it.name}”`);
      renderShop(card);
    } catch (e) { toast(e.message, true); }
  };
  return el("tr", {},
    el("td", {}, it.name),
    el("td", { class: "muted" }, it.description || "—"),
    el("td", { style: "text-align:right" }, fmtNum(it.price)),
    el("td", {}, roleChip(it.role_id)),
    el("td", {}, it.stock < 0 ? el("span", { class: "badge" }, "Unlimited") : String(it.stock)),
    el("td", { style: "text-align:right;white-space:nowrap" }, edit, del));
}

// editShopItem swaps a card into an inline edit form for one item.
function editShopItem(card, it) {
  const form = shopItemForm(it);
  const save = el("button", { class: "btn btn-primary" }, "Save");
  const cancel = el("button", { class: "btn btn-ghost" }, "Cancel");
  cancel.onclick = () => renderShop(card);
  save.onclick = async () => {
    const body = form.read();
    if (!body.name) { toast("Item name is required", true); return; }
    save.disabled = true;
    try {
      await api("PATCH", `/api/guilds/${state.guildID}/economy/shop/${it.id}`, body);
      toast(`Saved “${body.name}”`);
      renderShop(card);
    } catch (e) {
      toast(e.message, true);
      save.disabled = false;
    }
  };
  card.replaceChildren(
    el("div", { class: "card-head" }, el("h2", {}, `Edit “${it.name}”`)),
    el("div", { class: "stack", style: "margin-top:16px" }, form.node,
      el("div", { class: "card-actions" }, save, cancel)));
}

async function pageEconomy(root) {
  await loadGuildData(state.guildID);
  const lb = el("div", { class: "card" });
  const shop = el("div", { class: "card" });
  root.replaceChildren(
    pageHead("Economy", "Net-worth leaderboard and shop management", "coins"),
    lb, shop);
  await Promise.all([renderEconLeaderboard(lb), renderShop(shop)]);
}

/* ------------------------------------------------- leveling console (Inc3) */

const LVL_LIMIT = 25;

async function renderLevelLeaderboard(card) {
  card.replaceChildren(spinner());
  let page;
  try {
    const qs = new URLSearchParams({ limit: String(LVL_LIMIT), offset: String(state.lvl.offset) });
    page = await api("GET", `/api/guilds/${state.guildID}/leveling/leaderboard?${qs}`);
  } catch (e) {
    card.replaceChildren(emptyState("trophy", "Couldn't load leaderboard", e.message));
    return;
  }
  card.replaceChildren(el("div", { class: "card-head" },
    el("h2", {}, "XP leaderboard"), el("div", { class: "sub" }, `${page.total} ranked`)));
  if (!page.members.length) {
    card.appendChild(emptyState("trophy", "No XP yet", "Members appear here once they earn XP."));
    return;
  }
  const body = el("tbody");
  page.members.forEach((m, i) => {
    body.appendChild(el("tr", {},
      el("td", { class: "muted" }, "#" + (state.lvl.offset + i + 1)),
      memberCell(m),
      el("td", {}, el("span", { class: "badge act-timeout" }, "Lvl " + m.level)),
      el("td", { style: "text-align:right" }, fmtNum(m.xp)),
      el("td", { style: "text-align:right" }, fmtNum(m.messages))));
  });
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {},
      el("th", {}, "Rank"), el("th", {}, "Member"), el("th", {}, "Level"),
      el("th", { style: "text-align:right" }, "XP"),
      el("th", { style: "text-align:right" }, "Messages"))),
    body)));
  card.appendChild(pager(state.lvl.offset, page.members.length, page.total, LVL_LIMIT,
    (o) => { state.lvl.offset = o; renderLevelLeaderboard(card); }));
}

async function renderRewards(card) {
  card.replaceChildren(spinner());
  let data;
  try {
    data = await api("GET", `/api/guilds/${state.guildID}/leveling/rewards`);
  } catch (e) {
    card.replaceChildren(emptyState("trophy", "Couldn't load rewards", e.message));
    return;
  }
  const rewards = data.rewards || [];

  const level = el("input", { class: "input", type: "number", min: "1", placeholder: "Level", style: "max-width:100px" });
  const role = roleSelect("", false);
  role.style.maxWidth = "200px";
  const add = el("button", { class: "btn btn-primary" }, "Set reward");
  add.onclick = async () => {
    const lv = Number(level.value);
    if (!lv || lv < 1) { toast("Enter a level of 1 or more", true); return; }
    if (!role.value) { toast("Pick a role", true); return; }
    add.disabled = true;
    try {
      await api("PUT", `/api/guilds/${state.guildID}/leveling/rewards/${lv}`, { role_id: role.value });
      toast(`Reward set for level ${lv}`);
      renderRewards(card);
    } catch (e) {
      toast(e.message, true);
      add.disabled = false;
    }
  };

  card.replaceChildren(
    el("div", { class: "card-head" }, el("h2", {}, "Level rewards"),
      el("div", { class: "sub" }, "Grant a role when a member reaches a level.")),
    el("div", { class: "toolbar", style: "margin-top:16px" },
      el("label", { class: "dur" }, el("span", { class: "muted" }, "Level"), level),
      el("label", { class: "dur" }, el("span", { class: "muted" }, "Role"), role), add));

  if (!rewards.length) {
    card.appendChild(emptyState("trophy", "No rewards", "Add a level reward above."));
    return;
  }
  const body = el("tbody");
  for (const rw of rewards) {
    const del = el("button", { class: "btn btn-ghost btn-sm btn-danger", title: "Remove" }, icon("ban"));
    del.onclick = async () => {
      if (!(await confirmDialog("Remove reward?", `Clear the role reward at level ${rw.level}.`, true))) return;
      try {
        await api("DELETE", `/api/guilds/${state.guildID}/leveling/rewards/${rw.level}`);
        toast(`Removed level ${rw.level} reward`);
        renderRewards(card);
      } catch (e) { toast(e.message, true); }
    };
    body.appendChild(el("tr", {},
      el("td", {}, el("span", { class: "badge act-timeout" }, "Lvl " + rw.level)),
      el("td", {}, roleChip(rw.role_id)),
      el("td", { style: "text-align:right" }, del)));
  }
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {}, el("th", {}, "Level"), el("th", {}, "Role"), el("th", {}, ""))),
    body)));
}

async function pageLeveling(root) {
  await loadGuildData(state.guildID);
  const lb = el("div", { class: "card" });
  const rw = el("div", { class: "card" });
  root.replaceChildren(
    pageHead("Leveling", "XP leaderboard and level rewards", "trophy"),
    lb, rw);
  await Promise.all([renderLevelLeaderboard(lb), renderRewards(rw)]);
}

/* -------------------------------------------------- tickets console (Inc4) */

const TICKET_LIMIT = 25;

function ticketStatusBadge(s) {
  const cls = { open: "tk-open", claimed: "tk-claimed", closed: "tk-closed" }[s] || "";
  const label = { open: "Open", claimed: "Claimed", closed: "Closed" }[s] || s;
  return el("span", { class: "badge " + cls }, label);
}

// reasonDialog shows a modal with a single text field and resolves the trimmed
// value, or null when cancelled.
function reasonDialog(title, message, placeholder, danger) {
  return new Promise((resolve) => {
    const input = el("input", { class: "input", type: "text", placeholder, style: "width:100%;margin-top:12px" });
    const close = (v) => { overlay.remove(); resolve(v); };
    const overlay = el("div", { class: "modal-overlay" },
      el("div", { class: "modal" },
        el("h3", {}, title),
        el("p", { class: "muted" }, message),
        input,
        el("div", { class: "modal-actions" },
          el("button", { class: "btn", onClick: () => close(null) }, "Cancel"),
          el("button", { class: "btn " + (danger ? "btn-danger" : "btn-primary"), onClick: () => close(input.value.trim()) }, "Confirm"))));
    overlay.onclick = (e) => { if (e.target === overlay) close(null); };
    input.onkeydown = (e) => { if (e.key === "Enter") close(input.value.trim()); };
    document.body.appendChild(overlay);
    setTimeout(() => input.focus(), 0);
  });
}

// transcriptModal renders a ticket's recent messages in a scrollable panel.
function transcriptModal(tr) {
  const lines = tr.lines || [];
  const log = el("div", { class: "transcript" });
  if (!lines.length) {
    log.appendChild(el("div", { class: "muted" }, "No messages captured."));
  } else {
    for (const ln of lines) {
      const when = new Date(ln.timestamp);
      log.appendChild(el("div", { class: "tr-line" },
        el("span", { class: "tr-time muted mono" }, isNaN(when) ? "" : when.toLocaleTimeString()),
        el("span", { class: "tr-author" }, ln.author || "unknown"),
        el("span", { class: "tr-content" }, ln.content || "")));
    }
  }
  const close = () => overlay.remove();
  const overlay = el("div", { class: "modal-overlay" },
    el("div", { class: "modal modal-lg" },
      el("h3", {}, `Ticket #${tr.ticket.number} · transcript`),
      el("p", { class: "muted" }, tr.ticket.subject || ""),
      log,
      el("div", { class: "modal-actions" },
        el("button", { class: "btn btn-primary", onClick: close }, "Close"))));
  overlay.onclick = (e) => { if (e.target === overlay) close(); };
  document.body.appendChild(overlay);
}

function ticketRow(card, t) {
  const actions = [];
  if (t.status !== "closed") {
    const view = el("button", { class: "btn btn-ghost btn-sm", title: "View transcript" }, icon("scroll"));
    view.onclick = async () => {
      try {
        const tr = await api("GET", `/api/guilds/${state.guildID}/tickets/${t.id}/transcript`);
        transcriptModal(tr);
      } catch (e) { toast(e.message, true); }
    };
    actions.push(view);
    if (!t.claimer_id) {
      const claim = el("button", { class: "btn btn-ghost btn-sm", title: "Claim" }, icon("users"));
      claim.onclick = async () => {
        try {
          await api("POST", `/api/guilds/${state.guildID}/tickets/${t.id}/claim`);
          toast(`Claimed #${t.number}`);
          renderTickets(card);
        } catch (e) { toast(e.message, true); }
      };
      actions.push(claim);
    }
    const close = el("button", { class: "btn btn-ghost btn-sm btn-danger", title: "Close" }, icon("ban"));
    close.onclick = async () => {
      const reason = await reasonDialog(`Close ticket #${t.number}?`,
        "This deletes the channel and logs a transcript.", "Reason (optional)", true);
      if (reason === null) return;
      try {
        await api("POST", `/api/guilds/${state.guildID}/tickets/${t.id}/close`, { reason });
        toast(`Closed #${t.number}`);
        renderTickets(card);
      } catch (e) { toast(e.message, true); }
    };
    actions.push(close);
  }
  const when = new Date(t.created_at);
  return el("tr", {},
    el("td", {}, el("span", { class: "chip" }, "#" + t.number)),
    el("td", {}, t.subject || el("span", { class: "muted" }, "—")),
    el("td", {}, userCell(t.opener_name, t.opener_id)),
    el("td", {}, t.claimer_id ? userCell(t.claimer_name, t.claimer_id) : el("span", { class: "muted" }, "—")),
    el("td", {}, ticketStatusBadge(t.status)),
    el("td", { class: "muted" }, isNaN(when) ? "—" : when.toLocaleDateString()),
    el("td", { style: "text-align:right;white-space:nowrap" }, ...actions));
}

async function renderTickets(card) {
  card.replaceChildren(spinner());

  const mkTab = (key, label) => {
    const b = el("button", { class: "tab" + (state.tk.status === key ? " active" : "") }, label);
    b.onclick = () => {
      if (state.tk.status === key) return;
      state.tk.status = key;
      state.tk.offset = 0;
      renderTickets(card);
    };
    return b;
  };
  const tabs = el("div", { class: "tabs" }, mkTab("active", "Active"), mkTab("closed", "Closed"), mkTab("", "All"));

  let page;
  try {
    const qs = new URLSearchParams({ limit: String(TICKET_LIMIT), offset: String(state.tk.offset) });
    if (state.tk.status) qs.set("status", state.tk.status);
    page = await api("GET", `/api/guilds/${state.guildID}/tickets?${qs}`);
  } catch (e) {
    card.replaceChildren(emptyState("ticket", "Couldn't load tickets", e.message));
    return;
  }

  card.replaceChildren(
    el("div", { class: "card-head" }, el("h2", {}, "Tickets"),
      el("div", { class: "sub" }, `${page.total} total`)),
    el("div", { class: "toolbar", style: "margin-top:16px" }, tabs));

  if (!page.tickets.length) {
    card.appendChild(emptyState("ticket", "No tickets", "Tickets opened in this server show up here."));
    return;
  }
  const body = el("tbody");
  for (const t of page.tickets) body.appendChild(ticketRow(card, t));
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {},
      el("th", {}, "#"), el("th", {}, "Subject"), el("th", {}, "Opener"),
      el("th", {}, "Claimed by"), el("th", {}, "Status"), el("th", {}, "Opened"), el("th", {}, ""))),
    body)));
  card.appendChild(pager(state.tk.offset, page.tickets.length, page.total, TICKET_LIMIT,
    (o) => { state.tk.offset = o; renderTickets(card); }));
}

async function pageTickets(root) {
  await loadGuildData(state.guildID);
  const card = el("div", { class: "card" });
  root.replaceChildren(pageHead("Tickets", "Browse and manage support tickets", "ticket"), card);
  await renderTickets(card);
}

/* ------------------------------------------------ giveaways console (Inc4) */

const GW_LIMIT = 25;
const GW_UNITS = [
  { label: "minutes", ms: 60000 },
  { label: "hours", ms: 3600000 },
  { label: "days", ms: 86400000 },
];

function giveawayCreateCard(listCard) {
  const channel = channelSelect("");
  channel.style.maxWidth = "240px";
  const prize = el("input", { class: "input", type: "text", placeholder: "Prize", style: "flex:1;min-width:160px" });
  const winners = el("input", { class: "input", type: "number", min: "1", value: "1", style: "max-width:80px" });
  const durVal = el("input", { class: "input", type: "number", min: "1", value: "24", style: "max-width:80px" });
  const durUnit = el("select", { class: "input", style: "max-width:120px" },
    ...GW_UNITS.map((u, i) => {
      const o = el("option", { value: String(u.ms) }, u.label);
      if (i === 1) o.selected = true; // default to hours
      return o;
    }));

  const create = el("button", { class: "btn btn-primary" }, "Create");
  create.onclick = async () => {
    const body = {
      channel_id: channel.value,
      prize: prize.value.trim(),
      winners: Math.max(1, Number(winners.value) || 1),
      duration_ms: Math.max(1, Number(durVal.value) || 1) * Number(durUnit.value),
    };
    if (!body.channel_id) { toast("Pick a channel", true); return; }
    if (!body.prize) { toast("Prize is required", true); return; }
    create.disabled = true;
    try {
      await api("POST", `/api/guilds/${state.guildID}/giveaways`, body);
      toast(`Started giveaway for “${body.prize}”`);
      prize.value = "";
      state.gw.offset = 0;
      renderGiveaways(listCard);
    } catch (e) {
      toast(e.message, true);
    } finally {
      create.disabled = false;
    }
  };

  return el("div", { class: "card" },
    el("div", { class: "card-head" }, el("h2", {}, "New giveaway"),
      el("div", { class: "sub" }, "Posts an entry panel members join with a button.")),
    el("div", { class: "toolbar", style: "margin-top:16px" },
      el("label", { class: "dur" }, el("span", { class: "muted" }, "Channel"), channel),
      prize,
      el("label", { class: "dur" }, el("span", { class: "muted" }, "Winners"), winners),
      el("label", { class: "dur" }, el("span", { class: "muted" }, "Ends in"), durVal, durUnit),
      create));
}

function giveawayRow(card, g) {
  const actions = [];
  if (!g.ended) {
    const end = el("button", { class: "btn btn-ghost btn-sm", title: "End now" }, icon("clock"));
    end.onclick = async () => {
      if (!(await confirmDialog("End giveaway?", `Draw winners for “${g.prize}” now.`, false))) return;
      try {
        await api("POST", `/api/guilds/${state.guildID}/giveaways/${g.id}/end`);
        toast("Giveaway ended");
        renderGiveaways(card);
      } catch (e) { toast(e.message, true); }
    };
    actions.push(end);
  } else {
    const reroll = el("button", { class: "btn btn-ghost btn-sm", title: "Reroll winners" }, icon("rocket"));
    reroll.onclick = async () => {
      if (!(await confirmDialog("Reroll winners?", `Draw fresh winners for “${g.prize}”.`, false))) return;
      try {
        await api("POST", `/api/guilds/${state.guildID}/giveaways/${g.id}/reroll`, { winners: 0 });
        toast("Winners rerolled");
        renderGiveaways(card);
      } catch (e) { toast(e.message, true); }
    };
    actions.push(reroll);
  }

  let winnersCell;
  if (!g.ended) {
    winnersCell = el("td", { class: "muted" }, `${g.winners} target`);
  } else if (g.winner_ids && g.winner_ids.length) {
    winnersCell = el("td", {}, ...g.winner_ids.map((w) =>
      el("span", { class: "chip", style: "margin:0 4px 4px 0" }, w)));
  } else {
    winnersCell = el("td", { class: "muted" }, "No winners");
  }

  const ends = new Date(g.ends_at);
  return el("tr", {},
    el("td", {}, g.prize),
    el("td", { class: "muted" }, channelName(g.channel_id)),
    el("td", { style: "text-align:right" }, fmtNum(g.entries)),
    el("td", {}, g.ended
      ? el("span", { class: "badge tk-closed" }, "Ended")
      : el("span", { class: "badge tk-open" }, "Active")),
    winnersCell,
    el("td", { class: "muted" }, g.ended ? "—" : (isNaN(ends) ? "—" : ends.toLocaleString())),
    el("td", { style: "text-align:right;white-space:nowrap" }, ...actions));
}

async function renderGiveaways(card) {
  card.replaceChildren(spinner());
  let page;
  try {
    const qs = new URLSearchParams({ limit: String(GW_LIMIT), offset: String(state.gw.offset) });
    page = await api("GET", `/api/guilds/${state.guildID}/giveaways?${qs}`);
  } catch (e) {
    card.replaceChildren(emptyState("gift", "Couldn't load giveaways", e.message));
    return;
  }
  card.replaceChildren(el("div", { class: "card-head" }, el("h2", {}, "Giveaways"),
    el("div", { class: "sub" }, `${page.total} total`)));

  if (!page.giveaways.length) {
    card.appendChild(emptyState("gift", "No giveaways", "Create one above to get started."));
    return;
  }
  const body = el("tbody");
  for (const g of page.giveaways) body.appendChild(giveawayRow(card, g));
  card.appendChild(el("div", { class: "table-wrap" }, el("table", { class: "table" },
    el("thead", {}, el("tr", {},
      el("th", {}, "Prize"), el("th", {}, "Channel"),
      el("th", { style: "text-align:right" }, "Entries"), el("th", {}, "Status"),
      el("th", {}, "Winners"), el("th", {}, "Ends"), el("th", {}, ""))),
    body)));
  card.appendChild(pager(state.gw.offset, page.giveaways.length, page.total, GW_LIMIT,
    (o) => { state.gw.offset = o; renderGiveaways(card); }));
}

async function pageGiveaways(root) {
  await loadGuildData(state.guildID);
  const listCard = el("div", { class: "card" });
  root.replaceChildren(
    pageHead("Giveaways", "Create and manage prize draws", "gift"),
    giveawayCreateCard(listCard), listCard);
  await renderGiveaways(listCard);
}

/* ------------------------------------------------------------------ router */

function currentRoute() {
  const h = (location.hash || "#/overview").replace(/^#\/?/, "");
  const parts = h.split("/").filter(Boolean);
  if (parts[0] === "m" && parts[1]) return { kind: "module", name: parts[1] };
  if (parts[0] === "moderation") return { kind: "moderation" };
  if (parts[0] === "economy") return { kind: "economy" };
  if (parts[0] === "leveling") return { kind: "leveling" };
  if (parts[0] === "tickets") return { kind: "tickets" };
  if (parts[0] === "giveaways") return { kind: "giveaways" };
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
  if (route.kind === "moderation") return pageModeration(root);
  if (route.kind === "economy") return pageEconomy(root);
  if (route.kind === "leveling") return pageLeveling(root);
  if (route.kind === "tickets") return pageTickets(root);
  if (route.kind === "giveaways") return pageGiveaways(root);
  if (route.kind === "audit") return pageAudit(root);
  return pageOverview(root);
}

function markActiveNav(route) {
  for (const a of document.querySelectorAll("#nav a")) {
    const r = a.dataset.route;
    const active =
      (route.kind === "overview" && r === "overview") ||
      (route.kind === "audit" && r === "audit") ||
      (route.kind === "moderation" && r === "moderation") ||
      (route.kind === "economy" && r === "economy") ||
      (route.kind === "leveling" && r === "leveling") ||
      (route.kind === "tickets" && r === "tickets") ||
      (route.kind === "giveaways" && r === "giveaways") ||
      (route.kind === "module" && r === `m/${route.name}`);
    a.classList.toggle("active", active);
  }
}

/* ------------------------------------------------------------------- shell */

function navLink(route, label, ic) {
  return el("a", { dataset: { route }, href: `#/${route}` }, icon(ic), el("span", {}, label));
}

// CONSOLES are the management dashboards. Each shows only when its matching
// feature flag is true, so a console never appears without a backend seam.
const CONSOLES = [
  { key: "moderation", route: "moderation", label: "Moderation", icon: "shield" },
  { key: "economy", route: "economy", label: "Economy", icon: "coins" },
  { key: "leveling", route: "leveling", label: "Leveling", icon: "trophy" },
  { key: "tickets", route: "tickets", label: "Tickets", icon: "ticket" },
  { key: "giveaways", route: "giveaways", label: "Giveaways", icon: "gift" },
];

async function buildNav() {
  const nav = $("#nav");
  nav.replaceChildren(navLink("overview", "Overview", "home"));

  const active = CONSOLES.filter((c) => state.features[c.key]);
  if (active.length) {
    nav.appendChild(el("div", { class: "nav-label" }, "Management"));
    for (const c of active) nav.appendChild(navLink(c.route, c.label, c.icon));
  }

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
  state.mod = { target: "", action: "", offset: 0 };
  state.econ = { offset: 0 };
  state.lvl = { offset: 0 };
  state.tk = { status: "active", offset: 0 };
  state.gw = { offset: 0 };
  try {
    state.features = await api("GET", `/api/guilds/${id}/features`);
  } catch {
    state.features = {};
  }
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
