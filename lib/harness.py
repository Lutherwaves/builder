#!/usr/bin/env python3
"""Which agent harness runs builder, and where that harness keeps its files."""
from pathlib import Path

HARNESSES = ("claude", "codex", "cursor", "opencode")

# (override env var, default dir under home, plugin-root env var, process match)
_SPEC = {
    "claude": ("CLAUDE_CONFIG_DIR", (".claude",), "CLAUDE_PLUGIN_ROOT", (("x", "claude", None),)),
    "codex": ("CODEX_HOME", (".codex",), "CLAUDE_PLUGIN_ROOT", (("x", "codex", "app-server"),)),
    "cursor": (
        "CURSOR_CONFIG_DIR", (".cursor",), "CURSOR_PLUGIN_ROOT",
        (("f", "/cursor-agent/versions/[^/]+/index\\.js", "worker-server"),),
    ),
    "opencode": ("OPENCODE_CONFIG_DIR", (".config", "opencode"), None, (("x", "opencode", None),)),
}

# Cursor first: a Cursor hook started from a Claude Code session inherits CLAUDE* vars.
# Codex and opencode have no verified env marker; they arrive via explicit or BUILDER_HARNESS.
_ENV_MARKERS = (
    ("cursor", ("CURSOR_VERSION", "CURSOR_PLUGIN_ROOT")),
    ("claude", ("CLAUDECODE", "CLAUDE_PLUGIN_ROOT")),
)


def detect(env, explicit=None):
    for name in (explicit, env.get("BUILDER_HARNESS")):
        if name in HARNESSES:
            return name
    for name, markers in _ENV_MARKERS:
        if any(env.get(m) for m in markers):
            return name
    return "unknown"


def builder_home(env, home):
    override = env.get("BUILDER_HOME")
    return Path(override) if override else Path(home) / ".config" / "builder"


def paths(harness, env, home):
    if harness not in _SPEC:
        raise ValueError("unknown harness: %r" % (harness,))
    home = Path(home)
    override, default, root_var, match = _SPEC[harness]
    config_dir = Path(env[override]) if env.get(override) else home.joinpath(*default)
    p = {
        "config_dir": config_dir,
        "skills_dirs": [config_dir / "skills"],
        "plugins_record": None,
        "settings": None,
        "process_match": match,
        "plugin_root_var": root_var,
    }
    if harness == "claude":
        state = (config_dir if env.get(override) else home) / ".claude.json"
        p["mcp"] = [(state, "mcpServers")]
        p["plugins_record"] = config_dir / "plugins" / "installed_plugins.json"
        p["settings"] = config_dir / "settings.json"
    elif harness == "codex":
        p["mcp"] = [(config_dir / "config.toml", "mcp_servers")]
        p["plugins_record"] = config_dir / "config.toml"
        p["skills_dirs"].append(home / ".agents" / "skills")
    elif harness == "cursor":
        p["mcp"] = [(config_dir / "mcp.json", "mcpServers")]
        p["settings"] = config_dir / "hooks.json"
    else:
        p["mcp"] = [(config_dir / "opencode.json", "mcp")]
        p["plugins_record"] = config_dir / "opencode.json"
    return p


def all_process_match():
    return tuple(m for spec in _SPEC.values() for m in spec[3])
