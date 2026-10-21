import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import * as mod from "../.opencode/plugins/builder.js";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const soul = fs.readFileSync(path.join(root, "skills/builder/soul-compact.md"), "utf8");

let n = 0;
const sid = () => `ses_test_${++n}`;
const fakeClient = (parentID) => ({
  session: {
    get: async ({ path: { id } }) => ({ data: { id, ...(parentID ? { parentID } : {}) } }),
  },
});
const messages = (sessionID, text = "hello") => [
  { info: { role: "user", sessionID }, parts: [{ type: "text", text }] },
];
const spawnCounter = (result) => {
  const calls = [];
  const fn = (cmd, args, opts) => {
    calls.push({ cmd, args, opts });
    return result;
  };
  return { fn, calls };
};
const ok = (stdout) => ({ status: 0, stdout, error: undefined });
const transform = async (plugin, msgs) => {
  const output = { messages: msgs };
  await plugin["experimental.chat.messages.transform"]({}, output);
  return output.messages;
};

test("injects_once_per_session", async () => {
  const { fn, calls } = spawnCounter(ok("builder root: /x\nhello soul"));
  mod._setSpawn(fn);
  const plugin = await mod.BuilderPlugin({ client: fakeClient(), directory: root });
  const id = sid();
  const first = await transform(plugin, messages(id));
  const second = await transform(plugin, messages(id));
  assert.equal(calls.length, 1);
  assert.equal(calls[0].cmd, "python3");
  assert.deepEqual(calls[0].args.slice(1), ["--harness", "opencode", "--format", "text"]);
  assert.ok(calls[0].args[0].endsWith(path.join("hooks", "session_start.py")));
  assert.equal(calls[0].opts.timeout, 1000);
  assert.equal(first[0].parts.length, 2);
  assert.match(first[0].parts[0].text, /hello soul/);
  assert.equal(second[0].parts.length, 2);
  assert.match(second[0].parts[0].text, /hello soul/);
});

test("skips_child_session", async () => {
  const { fn, calls } = spawnCounter(ok("builder root: /x"));
  mod._setSpawn(fn);
  const plugin = await mod.BuilderPlugin({ client: fakeClient("ses_parent"), directory: root });
  const out = await transform(plugin, messages(sid()));
  assert.equal(out[0].parts.length, 1);
  assert.equal(calls.length, 0);
});

test("skips_when_marker_already_present", async () => {
  const { fn, calls } = spawnCounter(ok("builder root: /x"));
  mod._setSpawn(fn);
  const plugin = await mod.BuilderPlugin({ client: fakeClient(), directory: root });
  const out = await transform(plugin, messages(sid(), "builder root: /x\nrest"));
  assert.equal(out[0].parts.length, 1);
  assert.equal(calls.length, 0);
});

test("falls_back_to_soul_without_python", async () => {
  mod._setSpawn(() => ({ error: new Error("ENOENT"), status: null, stdout: "" }));
  const plugin = await mod.BuilderPlugin({ client: fakeClient(), directory: root });
  const out = await transform(plugin, messages(sid()));
  assert.equal(out[0].parts.length, 2);
  assert.equal(out[0].parts[0].text.trim(), soul.trim());
});

test("falls_back_to_soul_on_failed_exit_or_empty_output", async () => {
  for (const result of [{ status: 1, stdout: "boom" }, { status: 0, stdout: "  \n" }]) {
    mod._setSpawn(() => result);
    const plugin = await mod.BuilderPlugin({ client: fakeClient(), directory: root });
    const out = await transform(plugin, messages(sid()));
    assert.equal(out[0].parts[0].text.trim(), soul.trim());
  }
});

test("registers_skills_path_v1", async () => {
  const plugin = await mod.BuilderPlugin({ client: fakeClient(), directory: root });
  const config = {};
  await plugin.config(config);
  await plugin.config(config);
  assert.deepEqual(config.skills.paths, [path.join(root, "skills")]);
});

test("v2_setup_registers_every_skill", async () => {
  let added = [];
  let contextHook;
  const ctx = {
    skill: { transform: async (fn) => fn({ add: (s) => added.push(s) }) },
    session: { hook: async (name, fn) => { if (name === "context") contextHook = fn; } },
  };
  await mod.default.setup(ctx);
  const dirs = fs.readdirSync(path.join(root, "skills"), { withFileTypes: true })
    .filter((e) => e.isDirectory() && fs.existsSync(path.join(root, "skills", e.name, "SKILL.md")))
    .map((e) => e.name).sort();
  assert.deepEqual(added.map((s) => s.id).sort(), dirs);
  assert.ok(added.every((s) => s.path.endsWith("SKILL.md") && s.content && s.name));
  assert.equal(typeof contextHook, "function");

  const { fn } = spawnCounter(ok("builder root: /x\nv2 soul"));
  mod._setSpawn(fn);
  const event = { sessionID: sid(), messages: [{ role: "user", content: [{ type: "text", text: "hi" }] }] };
  await contextHook(event);
  assert.match(event.messages[0].content[0].text, /v2 soul/);
});

test("v2_setup_ignores_v1_shaped_ctx", async () => {
  await mod.default.setup({});
});

test("default_export_shape", () => {
  assert.equal(mod.default.id, "builder");
  assert.equal(mod.default.server, mod.BuilderPlugin);
  assert.equal(typeof mod.default.setup, "function");
});

test("string_exports_absent", () => {
  for (const [key, value] of Object.entries(mod)) {
    if (key === "default") continue;
    assert.equal(typeof value, "function", `export ${key} must be a function`);
  }
});
