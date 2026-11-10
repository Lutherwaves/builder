The classes of these lists follow cloudflare/security-audit-skill (MIT). No text is copied from it.

# security — domain checklists

The `security` skill loads only the lists that fit the repository. Each line is one check.
The name in backticks is the dimension that a finding from the check goes under.
A check that depends on a setting outside the repository gives a `needs_validation` record, never a guess.

## AI, LLM and agents

- Text from a web page, a file, an email or a tool result reaches the prompt with no boundary between data and instructions. `agent-boundary`
- A tool can delete, pay, send or deploy, and nothing asks a person before the call. `agent-boundary`
- A tool has wider scopes than its task: a write token where a read token is enough, or a shell where one command is enough. `agent-boundary`
- Model output becomes a shell command, a SQL query, a file path or a URL with no check. `injection`
- Retrieval or memory returns one tenant's documents to another tenant. `authz`
- An agent can write to its own long-term memory from untrusted input, so the next session obeys it. `agent-boundary`
- An MCP server or plugin comes from an unpinned source, or its tool descriptions can change after install. `supply-chain`
- A subagent inherits every tool and secret of its parent. `agent-boundary`
- The system prompt, a key or another user's history can leak into a reply. `secrets`
- A model reply is rendered as HTML or markdown with live links and images, so it can carry data out. `injection`
- One request can start an unbounded loop of model or tool calls that bills the owner. `availability`
- The tool output that a person sees differs from what the agent acted on. `data-integrity`

## Cloud and deployment

- A workload identity or service account can reach more than its own resources. `authz`
- A storage bucket, a database or an admin console is reachable from the internet. `osi-l7`
- The metadata endpoint of the instance is reachable through an SSRF or a proxy route. `ssrf`
- A container runs as root, privileged, or with the host network, the docker socket or a host path mounted. `osi-l7`
- A secret sits in an image layer, a build argument, a plain environment file or a Terraform state. `secrets`
- The ingress, the load balancer or a security group exposes a port that only the inside needs. `osi-l3-l4`
- An origin server accepts traffic that skips the CDN or the proxy that holds its limits. `osi-l3-l4`
- A queue, a bucket event or a scheduler can trigger a function with no check of the caller. `authn`
- A preview, staging or debug deployment shares data or secrets with production. `secrets`
- Autoscaling has no ceiling, or its ceiling is above the quota of a paid downstream. `availability`
- A deploy role in CI can change IAM or delete production resources. `supply-chain`
- A config key turns off auth or TLS, and its default is the unsafe value. `osi-l7`

## Client side

- Data from the URL, storage or a message reaches `innerHTML`, `eval` or a script URL. `injection`
- A `postMessage` handler does not check the sender's origin, or a sender posts to `*`. `authz`
- A CORS policy reflects any origin and allows credentials. `authz`
- A token or personal data sits in `localStorage` where any injected script can read it. `secrets`
- A service worker caches an authenticated response and serves it to the next user of the device. `secrets`
- A page that changes state can be framed, so a click on it can be stolen. `authn`
- An open redirect or a `javascript:` link comes from input. `injection`
- A third-party script loads with no integrity hash, or from a domain that the team does not own. `supply-chain`
- A response size, a timing or a cache state tells another site whether a user is signed in. `secrets`
- The client enforces a price, a role or a limit that the server does not check again. `authz`
- A prototype key from input (`__proto__`, `constructor`) reaches a deep merge or an object path setter. `injection`

## Desktop, mobile and local IPC

- A deep link or a custom URL scheme starts an action with no confirmation by the user. `authn`
- An OAuth redirect goes to a scheme or a loopback port that another local app can claim. `authn`
- A webview loads remote content and exposes a native bridge to it. `agent-boundary`
- A local HTTP server, socket or named pipe accepts calls from any local process or any web page. `authn`
- An exported component, intent or XPC service runs actions for a caller that it does not check. `authz`
- A privileged helper runs a path, an argument or a script that a normal user can change. `injection`
- A file in a world-writable or temp directory is opened by a privileged process (a link or race attack). `data-integrity`
- A token sits in plain preferences instead of the keychain or the keystore. `secrets`
- An update downloads over plain HTTP, or installs a package with no signature check. `supply-chain`
- A backup, a log or a screenshot of the app carries secrets or personal data. `secrets`
- The app trusts a debug flag, a root check or a pin that a user of the device can turn off. `data-integrity`

## Protocols, RPC and messaging

- Two parsers read the same message differently: a proxy and an app, a header and a body, or two JSON libraries. `data-integrity`
- A message has no size, depth or count limit before the server decodes it. `memory`
- An RPC method has no authorization check of its own because the gateway is assumed to check it. `authz`
- A field in the message (a user id, a tenant id, a role) is trusted as the caller's identity. `authz`
- A consumer reads a queue or a topic that other tenants or services can write to. `authz`
- A signed message can be replayed: no nonce, no timestamp window, no idempotency key. `data-integrity`
- Messages that arrive out of order or twice can move money or state twice. `data-integrity`
- A dead-letter or retry queue runs a poisoned message again and again. `availability`
- A WebSocket or a stream checks auth once at connect and never again. `authn`
- A gRPC or GraphQL reflection, introspection or debug endpoint is open in production. `osi-l7`
- A GraphQL query or batch has no cost or depth limit. `availability`
- A broker or an internal RPC call runs with no TLS or with certificate checks off. `osi-l5-l6`
