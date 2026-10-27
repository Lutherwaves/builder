/**
 * builder plugin for OpenCode.
 *
 * V1 (opencode 1.x): named export BuilderPlugin. The config hook registers the
 * skills directory. experimental.chat.messages.transform injects the builder
 * soul once into the first user message of a top-level session.
 *
 * V2 (opencode 2.x): default export { id, server, setup }. setup() registers
 * every skill with ctx.skill.transform and injects the soul with
 * ctx.session.hook("context").
 *
 * No dependencies. The only exports are BuilderPlugin and the default object: older 1.x hosts call each export.
 */

import path from "path";
import fs from "fs";
import { spawnSync } from "child_process";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, "../..");
const skillsDir = path.join(root, "skills");
const MARKER = "builder root:";

// Test seam: tests set globalThis.__builderSpawn. It is not an export, because opencode 1.x calls every export as a plugin factory.
const spawn = (...args) =>
  (typeof globalThis.__builderSpawn === "function" ? globalThis.__builderSpawn : spawnSync)(...args);

const readSoul = () => {
  try {
    return fs.readFileSync(path.join(skillsDir, "builder", "soul-compact.md"), "utf8");
  } catch {
    return null;
  }
};

const runHook = () => {
  try {
    const result = spawn(
      path.join(root, "hooks", "session-start"),
      ["--harness", "opencode", "--format", "text"],
      { timeout: 1000, encoding: "utf8" },
    );
    if (result && !result.error && result.status === 0 && typeof result.stdout === "string" && result.stdout.trim()) {
      return result.stdout;
    }
  } catch {
    // fall through to the soul
  }
  return readSoul();
};

// sessionID -> bootstrap text. The hook fires on every agent step, so run the hook once per session.
const CACHE_MAX = 512;
const _bootstrapCache = new Map();

const bound = (cache) => {
  if (cache.size < CACHE_MAX) return;
  let toDrop = Math.ceil(CACHE_MAX / 4);
  for (const key of cache.keys()) {
    if (toDrop-- <= 0) break;
    cache.delete(key);
  }
};

const getBootstrap = (sessionID) => {
  if (sessionID && _bootstrapCache.has(sessionID)) return _bootstrapCache.get(sessionID);
  const text = runHook();
  if (sessionID) {
    bound(_bootstrapCache);
    _bootstrapCache.set(sessionID, text);
  }
  return text;
};

// A session with a parentID is a subagent session: it does not get the bootstrap.
const _childSessionCache = new Map();

const isChildSession = async (fetchSession, sessionID) => {
  if (!sessionID) return false;
  if (_childSessionCache.has(sessionID)) return _childSessionCache.get(sessionID);

  let isChild = false;
  try {
    const result = await fetchSession(sessionID);
    // V1 returns an SDK envelope, V2 a direct session record.
    if (!result || typeof result !== "object" || Array.isArray(result)) {
      throw new Error("Session lookup returned no usable record");
    }
    if (result.error != null || result.response?.ok === false) {
      throw new Error("Session lookup was unsuccessful");
    }
    const session = "data" in result ? result.data : result;
    if (!session || typeof session !== "object" || Array.isArray(session) || session.id !== sessionID) {
      throw new Error("Session lookup returned an invalid session identity");
    }
    if (session.parentID !== undefined && (typeof session.parentID !== "string" || session.parentID.length === 0)) {
      throw new Error("Session lookup returned an invalid parent identity");
    }
    isChild = session.parentID !== undefined;
  } catch (err) {
    // Fail open and do not cache, so a transient failure can recover on the next step.
    console.error("[builder] session lookup failed, treating session as top-level:", err);
    return false;
  }
  bound(_childSessionCache);
  _childSessionCache.set(sessionID, isChild);
  return isChild;
};

const extractFrontmatter = (content) => {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!match) return { frontmatter: {}, body: content };
  const frontmatter = {};
  let lastKey = null;
  for (const rawLine of match[1].split("\n")) {
    const line = rawLine.replace(/\r$/, "");
    const colonIdx = line.indexOf(":");
    if (colonIdx > 0 && !/^\s/.test(line)) {
      lastKey = line.slice(0, colonIdx).trim();
      const value = line.slice(colonIdx + 1).trim();
      frontmatter[lastKey] = /^(>[+-]?|\|[+-]?)$/.test(value) ? "" : value;
    } else if (lastKey !== null && line.trim() !== "") {
      frontmatter[lastKey] = `${frontmatter[lastKey]} ${line.trim()}`.trim();
    }
  }
  for (const key of Object.keys(frontmatter)) {
    frontmatter[key] = frontmatter[key].replace(/^(["'])([\s\S]*)\1$/, "$2");
  }
  return { frontmatter, body: match[2] };
};

export const BuilderPlugin = async ({ client }) => ({
  config: async (config) => {
    // V2: skills is a flat array and setup() registers them.
    if (Array.isArray(config.skills)) return;
    config.skills = config.skills || {};
    config.skills.paths = config.skills.paths || [];
    if (!config.skills.paths.includes(skillsDir)) config.skills.paths.push(skillsDir);
  },

  "experimental.chat.messages.transform": async (_input, output) => {
    if (!output.messages.length) return;
    const firstUser = output.messages.find((m) => m.info.role === "user");
    if (!firstUser || !firstUser.parts.length) return;
    if (firstUser.parts.some((p) => p.type === "text" && p.text.includes(MARKER))) return;
    // V1 passes no input to this hook, so take the sessionID from the message.
    const sessionID = firstUser.info.sessionID;
    if (client && (await isChildSession((id) => client.session.get({ path: { id } }), sessionID))) return;
    const bootstrap = getBootstrap(sessionID);
    if (!bootstrap) return;
    firstUser.parts.unshift({ ...firstUser.parts[0], type: "text", text: bootstrap });
  },
});

async function setup(ctx) {
  // V1 also calls default.setup, with a ctx that lacks the skill and session domains.
  if (!ctx || !ctx.skill || typeof ctx.skill.transform !== "function" || !ctx.session || typeof ctx.session.hook !== "function") {
    return;
  }

  try {
    const skills = [];
    if (fs.existsSync(skillsDir)) {
      for (const entry of fs.readdirSync(skillsDir, { withFileTypes: true })) {
        if (!entry.isDirectory() || entry.name.startsWith(".")) continue;
        const skillPath = path.join(skillsDir, entry.name, "SKILL.md");
        if (!fs.existsSync(skillPath)) continue;
        const { frontmatter, body } = extractFrontmatter(fs.readFileSync(skillPath, "utf8"));
        skills.push({
          id: entry.name,
          name: frontmatter.name || entry.name,
          ...(frontmatter.description ? { description: frontmatter.description } : {}),
          path: skillPath,
          content: body,
        });
      }
    }
    await ctx.skill.transform((draft) => {
      // A throw escaping this callback disables the whole plugin, so contain each skill.
      for (const skill of skills) {
        try {
          draft.add(skill);
        } catch (err) {
          console.error(`[builder] skill "${skill.id}" rejected by host, skipping:`, err);
        }
      }
    });
  } catch (err) {
    console.error("[builder] skill registration failed:", err);
  }

  try {
    await ctx.session.hook("context", async (event) => {
      try {
        if (!event.messages || !event.messages.length) return;
        const firstUser = event.messages.find((m) => m.role === "user");
        if (firstUser && (!firstUser.content || !firstUser.content.length)) return;
        if (firstUser?.content.some((p) => p.type === "text" && p.text && p.text.includes(MARKER))) return;
        if (
          typeof ctx.session.get === "function" &&
          (await isChildSession((id) => ctx.session.get({ sessionID: id }), event.sessionID))
        ) return;
        const bootstrap = getBootstrap(event.sessionID);
        if (!bootstrap) return;
        if (firstUser) {
          firstUser.content.unshift({ type: "text", text: bootstrap });
        } else {
          event.messages.push({ role: "user", content: [{ type: "text", text: bootstrap }] });
        }
      } catch (err) {
        console.error("[builder] context hook failed:", err);
      }
    });
  } catch (err) {
    console.error("[builder] session hook registration failed:", err);
  }
}

export default {
  id: "builder",
  server: BuilderPlugin,
  setup,
};
