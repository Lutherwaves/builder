# security — standards map

The `security` skill reads this file when it maps a finding. Each identifier here comes from its source. Never add one from memory.
If a finding fits no row, find its CWE at <https://cwe.mitre.org> and say "no row in standards.md" in the finding.

## Editions

| Standard | Edition | Source |
|---|---|---|
| OWASP Top 10 | 2025 | <https://top10.owasp.org/2025> |
| OWASP API Security Top 10 | 2023 | <https://owasp.org/API-Security/editions/2023/en/0x11-t10/> |
| OWASP ASVS | 5.0 | <https://github.com/OWASP/ASVS/tree/master/5.0/en> |
| CWE Top 25 | 2025 | `Top25_2025` in `internal/security/security.go`. The `score` command sets the flag |
| CAPEC | current | <https://capec.mitre.org> |
| Bugcrowd VRT | release of 2026-07-08 | <https://github.com/bugcrowd/vulnerability-rating-taxonomy> |
| NIST SP 800-53 | Rev. 5 | <https://csrc.nist.gov/pubs/sp/800/53/r5/upd1/final> |
| CIS Controls | v8.1 | <https://www.cisecurity.org/controls/v8-1> |

## The map

Dimension names are the values `builder security` accepts. A dash means that the standard has no fitting entry.
A † marks a Top 10:2025 category that fits by theme only: the row's CWE is in no 2025 category's mapped CWE list. Say "by theme" when you cite it.
Each CWE here is ALLOWED or ALLOWED-WITH-REVIEW in MITRE's vulnerability-mapping notes. Do not map to a DISCOURAGED one, such as CWE-200, CWE-345 or CWE-400.

| Class | Dimension | CWE | CAPEC | Top 10:2025 | API 2023 | ASVS 5.0 | VRT | NIST 800-53 | CIS v8.1 |
|---|---|---|---|---|---|---|---|---|---|
| SQL injection | injection | CWE-89 | CAPEC-66 | A05 Injection | — | V1 Encoding and Sanitization | Server-Side Injection | SI-10 | 16.12 |
| OS command injection | injection | CWE-78 | CAPEC-88 | A05 Injection | — | V1 Encoding and Sanitization | Server-Side Injection | SI-10 | 16.12 |
| Code injection, unsafe eval | injection | CWE-94 | CAPEC-242 | A05 Injection | — | V1 Encoding and Sanitization | Server-Side Injection | SI-10 | 16.12 |
| Cross-site scripting | injection | CWE-79 | CAPEC-63 | A05 Injection | — | V3 Web Frontend Security | Cross-Site Scripting (XSS) | SI-10 | 16.12 |
| LLM prompt injection | agent-boundary | CWE-1427 | — | A05 Injection | — | — | AI Application Security | SI-10 | 16.10 |
| Path traversal | authz | CWE-22 | CAPEC-126 | A01 Broken Access Control | — | V5 File Handling | Server Security Misconfiguration | AC-3 | 16.12 |
| Missing authorization | authz | CWE-862 | CAPEC-122 | A01 Broken Access Control | API5 Broken Function Level Authorization | V8 Authorization | Broken Access Control (BAC) | AC-3 | 6.8 |
| Object access by a user-controlled key (IDOR, BOLA) | authz | CWE-639 | CAPEC-122 | A01 Broken Access Control | API1 Broken Object Level Authorization | V8 Authorization | Broken Access Control (BAC) | AC-3 | 6.8 |
| Tenant isolation gap | authz | CWE-863 | CAPEC-122 | A01 Broken Access Control | API1 Broken Object Level Authorization | V8 Authorization | Broken Access Control (BAC) | AC-4 | 6.8 |
| Cross-site request forgery | authn | CWE-352 | CAPEC-62 | A01 Broken Access Control | — | V3 Web Frontend Security | Cross-Site Request Forgery (CSRF) | SC-23 | 16.12 |
| Server-side request forgery | ssrf | CWE-918 | CAPEC-664 | A01 Broken Access Control | API7 Server Side Request Forgery | V15 Secure Coding and Architecture | Server Security Misconfiguration | SC-7 | 13.10 |
| Missing authentication on a critical function | authn | CWE-306 | CAPEC-115 | A07 Authentication Failures | API2 Broken Authentication | V6 Authentication | Broken Authentication and Session Management | IA-2 | 6.3 |
| Session fixation or a weak session | authn | CWE-384 | CAPEC-61 | A07 Authentication Failures | API2 Broken Authentication | V7 Session Management | Broken Authentication and Session Management | SC-23 | 16.12 |
| Unsigned webhook or data of unverified origin | data-integrity | CWE-347 | — | A08 Software or Data Integrity Failures † | API10 Unsafe Consumption of APIs | V4 API and Web Service | Broken Authentication and Session Management | SI-7 | 16.10 |
| Unsafe deserialization | injection | CWE-502 | CAPEC-586 | A08 Software or Data Integrity Failures | — | V1 Encoding and Sanitization | Server-Side Injection | SI-10 | 16.12 |
| Race or check-then-act | data-integrity | CWE-362 | CAPEC-26 | A06 Insecure Design | API6 Unrestricted Access to Sensitive Business Flows | V2 Validation and Business Logic | Server Security Misconfiguration | — | 16.10 |
| Hard-coded or committed secret | secrets | CWE-798 | CAPEC-191 | A07 Authentication Failures | — | V13 Configuration | Sensitive Data Exposure | IA-5 | 16.12 |
| Secret or personal data in a log | secrets | CWE-532 | — | A09 Security Logging and Alerting Failures | — | V16 Security Logging and Error Handling | Sensitive Data Exposure | AU-3 | 3.3 |
| Sensitive data exposed to the wrong actor | secrets | CWE-201 | CAPEC-118 | A01 Broken Access Control † | API3 Broken Object Property Level Authorization | V14 Data Protection | Sensitive Data Exposure | AC-3 | 3.3 |
| TLS certificate check off | osi-l5-l6 | CWE-295 | CAPEC-94 | A07 Authentication Failures | API8 Security Misconfiguration | V12 Secure Communication | Insecure Data Transport | SC-8 | 3.10 |
| Cleartext transport of sensitive data | osi-l5-l6 | CWE-319 | CAPEC-157 | A04 Cryptographic Failures | API8 Security Misconfiguration | V12 Secure Communication | Insecure Data Transport | SC-8 | 3.10 |
| Weak or broken cryptography | osi-l5-l6 | CWE-327 | CAPEC-97 | A04 Cryptographic Failures | — | V11 Cryptography | Cryptographic Weakness | SC-13 | 3.10 |
| Unbounded read or allocation | memory | CWE-770 | CAPEC-130 | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V2 Validation and Business Logic | Application-Level Denial-of-Service (DoS) | SC-5 | 16.10 |
| Decompression or archive bomb | memory | CWE-409 | CAPEC-130 | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V5 File Handling | Application-Level Denial-of-Service (DoS) | SC-5 | 16.10 |
| XML entity expansion | memory | CWE-776 | CAPEC-197 | A02 Security Misconfiguration | API4 Unrestricted Resource Consumption | V1 Encoding and Sanitization | Application-Level Denial-of-Service (DoS) | SC-5 | 16.12 |
| Regular expression denial of service | availability | CWE-1333 | CAPEC-492 | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V2 Validation and Business Logic | Application-Level Denial-of-Service (DoS) | SC-5 | 16.12 |
| Uncontrolled recursion | memory | CWE-674 | — | A10 Mishandling of Exceptional Conditions † | API4 Unrestricted Resource Consumption | V15 Secure Coding and Architecture | Application-Level Denial-of-Service (DoS) | SC-5 | 16.12 |
| Blocking work on an async event loop | availability | CWE-834 | — | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V15 Secure Coding and Architecture | Application-Level Denial-of-Service (DoS) | SC-5 | 16.10 |
| No rate or quota limit on a public or expensive endpoint | availability | CWE-770 | CAPEC-125 | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V2 Validation and Business Logic | Server Security Misconfiguration | SC-5 | 13.10 |
| Unlimited calls to a paid dependency (denial of wallet) | availability | CWE-770 | CAPEC-125 | A06 Insecure Design † | API4 Unrestricted Resource Consumption | V2 Validation and Business Logic | Application-Level Denial-of-Service (DoS) | SC-6 | 16.10 |
| No limit on login or code attempts | authn | CWE-307 | CAPEC-49 | A07 Authentication Failures | API2 Broken Authentication | V6 Authentication | Broken Authentication and Session Management | AC-7 | 6.3 |
| Business flow open to automation | availability | CWE-799 | — | A06 Insecure Design | API6 Unrestricted Access to Sensitive Business Flows | V2 Validation and Business Logic | Server Security Misconfiguration | SC-5 | 16.10 |
| CI runs untrusted code with secrets | supply-chain | CWE-829 | — | A08 Software or Data Integrity Failures | — | V15 Secure Coding and Architecture | Server Security Misconfiguration | SR-3 | 16.1 |
| Dependency with a known vulnerability | supply-chain | CWE-1395 | — | A03 Software Supply Chain Failures | API10 Unsafe Consumption of APIs | V15 Secure Coding and Architecture | Using Components with Known Vulnerabilities | RA-5 | 16.5 |
| Unpinned action, image or package | supply-chain | CWE-494 | CAPEC-538 | A08 Software or Data Integrity Failures | — | V15 Secure Coding and Architecture | Using Components with Known Vulnerabilities | SR-11 | 16.5 |
| Agent tool with more power than its task | agent-boundary | CWE-250 | — | A01 Broken Access Control † | API5 Broken Function Level Authorization | V8 Authorization | AI Application Security | AC-6 | 6.8 |
| Error path that fails open | data-integrity | CWE-636 | — | A10 Mishandling of Exceptional Conditions | — | V16 Security Logging and Error Handling | Server Security Misconfiguration | SI-11 | 16.10 |
| Missing security event log | data-integrity | CWE-778 | — | A09 Security Logging and Alerting Failures | — | V16 Security Logging and Error Handling | — | AU-2 | 8.2 |
| Debug mode, default credential or open admin in production | osi-l7 | CWE-1188 | — | A02 Security Misconfiguration † | API8 Security Misconfiguration | V13 Configuration | Server Security Misconfiguration | CM-6 | 4.1 |
| Edge with no DDoS or connection limits | osi-l3-l4 | CWE-770 | CAPEC-125 | A02 Security Misconfiguration † | API4 Unrestricted Resource Consumption | — | Network Security Misconfiguration | SC-5 | 13.10 |

## OSI layers, scoped honestly

| Layer | Usually owned by | What to check |
|---|---|---|
| L1 physical, L2 data link | The cloud or hosting provider | Name the provider and the shared-responsibility line. Do not invent a finding. On-premises or edge hardware: ask who owns the cabling, the switches and the Wi-Fi. |
| L3 network, L4 transport | The provider's edge, plus your firewall and load balancer config | Volumetric DDoS protection at the edge. Open ports and security groups. Connection, stream and slowloris limits. Server timeouts. Per-IP connection caps. |
| L5 session, L6 presentation | Your TLS terminator and your code | Who ends TLS. Minimum version. Certificate checks on every outbound call. HSTS. Cookie flags. Encodings and parsers that accept attacker input. |
| L7 application | You | Every other row of the map. This is the bulk of the review. |

## Memory safety and resource exhaustion, per language

| Language | Look for |
|---|---|
| Python | `read()` with no size, `json.loads` on an unbounded body, `zipfile`/`tarfile` `extractall` with no size or entry cap, `xml.etree` on untrusted XML (use `defusedxml`), `re` on a visitor's pattern, `pickle.loads`, PIL with no `MAX_IMAGE_PIXELS`, sync I/O inside `async def`. |
| Node / TypeScript | Body parsers with no `limit`, `JSON.parse` on an unbounded stream, sync `fs` or CPU work on the event loop, `RegExp` from input, `zlib` inflate with no `maxOutputLength`, unbounded `Promise.all` fan-out. |
| Go | `io.ReadAll` with no `http.MaxBytesReader` or `io.LimitReader`, a goroutine per request with no bound, `gzip`/`zip` readers with no size cap, an `http.Server` with no `ReadHeaderTimeout`, deep recursion in a parser. |
| Rust | `unsafe` blocks and their invariants, `Vec::with_capacity` from an input length, `read_to_end` with no limit, `serde` on deeply nested input with no recursion limit, blocking calls in an async task. |
| Java / JVM | XML parsers with external entities on, `ObjectInputStream`, `readAllBytes` with no limit, `ZipInputStream` with no cap, unbounded thread pools and queues. |
| C / C++ | Buffer bounds (CWE-120, CWE-121, CWE-122, CWE-787, CWE-125), use after free (CWE-416), NULL dereference (CWE-476), integer overflow in a size calculation (CWE-190), format strings (CWE-134), `alloca` or VLA sized by input. |
