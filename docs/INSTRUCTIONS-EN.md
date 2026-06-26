# Complete Guide to Configuring and Using S-UI-X Extended

This comprehensive guide details the architecture, parameters, and operational logic of the S-UI-X Extended web panel, which runs on the sing-box-extended core. It includes step-by-step instructions for configuring inbound and outbound connections, routing rules, DNS, bandwidth/connection limiters, and system services.

## Table of Contents

1. [System Topology and Object Dependencies](#1-system-topology-and-object-dependencies)
   * [Key Object References](#key-object-references)
2. [Inbounds (Incoming Connections)](#2-inbounds-incoming-connections)
   * [General Socket Settings (Listen Options)](#general-socket-settings-listen-options)
   * [Advanced Inbound Parameters](#advanced-inbound-parameters)
   * [Inbound Protocols](#inbound-protocols)
3. [Outbounds (Outgoing Connections)](#3-outbounds-outgoing-connections)
4. [Socket Connection Settings (Dial Options)](#4-socket-connection-settings-dial-options)
5. [Endpoints (Network Tunnels)](#5-endpoints-network-tunnels)
   * [WireGuard (Warp)](#wireguard-warp)
   * [Tailscale](#tailscale)
   * [VPN](#vpn)
6. [Services (System Processes)](#6-services-system-processes)
   * [resolved](#resolved)
   * [ssm-api](#ssm-api)
   * [derp](#derp)
   * [ocm / ccm](#ocm-ccm)
   * [oom-killer](#oom-killer)
   * [profiler](#profiler)
7. [TLS Settings](#7-tls-settings)
   * [Server TLS (Inbound Security)](#server-tls-inbound-security)
   * [Client TLS (Outbound Security)](#client-tls-outbound-security)
   * [Protocol Compatibility with TLS and Reality](#protocol-compatibility-with-tls-and-reality)
     * [Protocols Supporting Classic TLS](#protocols-supporting-classic-tls)
     * [Protocols Supporting Reality](#protocols-supporting-reality)
8. [Rules and Rule Sets (Traffic Routing)](#8-rules-and-rule-sets-traffic-routing)
   * [Rule Parameter Fields](#rule-parameter-fields)
9. [DNS Settings](#9-dns-settings)
   * [DNS Resolvers](#dns-resolvers)
   * [DNS Rules](#dns-rules)
10. [Load Balancing and Fallbacks](#10-load-balancing-and-fallbacks)
    * [Selector](#selector)
    * [URLTest](#urltest)
    * [Fallback](#fallback)
    * [Failover](#failover)
    * [Bond](#bond)
    * [CoreFailover](#corefailover)
    * [Parser](#parser)
11. [Traffic Limiters](#11-traffic-limiters)
12. [Step-by-Step Scenarios](#12-step-by-step-scenarios)
    * [Scenario 1: VLESS-Reality Setup (TLS camouflage)](#scenario-1-vless-reality-setup-tls-camouflage)
    * [Scenario 2: Shadowsocks Setup (Classic Fast Proxy)](#scenario-2-shadowsocks-setup-classic-fast-proxy)
    * [Scenario 3: ShadowTLS with Hidden Shadowsocks](#scenario-3-shadowtls-with-hidden-shadowsocks)
13. [System Maintenance](#13-system-maintenance)
    * [Subscription Security (Sub Secrets)](#subscription-security-sub-secrets)
    * [Client IP Monitoring](#client-ip-monitoring)
    * [Panel Updates and Database Backups](#panel-updates-and-database-backups)
14. [Troubleshooting](#14-troubleshooting)
15. [Protocol Comparison Matrix](#15-protocol-comparison-matrix)
16. [Glossary of Terms](#16-glossary-of-terms)
17. [Security and Testing Checklist](#17-security-and-testing-checklist)
18. [Differences Between Similar Network Mechanisms](#18-differences-between-similar-network-mechanisms)
    * [CoreFailover vs Failover (Panel-Managed)](#corefailover-vs-failover-panel-managed)
    * [Bond vs Selector / URLTest](#bond-vs-selector-urltest)
19. [Differences Between Inbound and Outbound Configurations](#19-differences-between-inbound-and-outbound-configurations)
20. [Detailed Protocol Configuration Procedures](#20-detailed-protocol-configuration-procedures)
    * [Inbound Configurations](#inbound-configurations)
    * [Outbound Configurations](#outbound-configurations)
21. [Detailed Endpoint Configuration Procedures](#21-detailed-endpoint-configuration-procedures)
    * [WireGuard (Warp)](#wireguard-warp-1)
    * [Tailscale](#tailscale-1)
    * [VPN](#vpn-1)
22. [Detailed Service Configuration Procedures](#22-detailed-service-configuration-procedures)
    * [resolved](#resolved-1)
    * [ssm-api](#ssm-api-1)
    * [derp](#derp-1)
    * [ccm / ocm](#ccm-ocm)
    * [oom-killer](#oom-killer-1)
    * [profiler](#profiler-1)
23. [Detailed TLS Settings Configuration Procedures](#23-detailed-tls-settings-configuration-procedures)
    * [Creating Server TLS Profiles (For Inbounds)](#creating-server-tls-profiles-for-inbounds)
    * [Creating Reality Profiles](#creating-reality-profiles)
24. [Detailed Rules and Rulesets Configuration Procedures](#24-detailed-rules-and-rulesets-configuration-procedures)
    * [Creating Routing Rules (Rule)](#creating-routing-rules-rule)
    * [Creating Rule Lists (Ruleset)](#creating-rule-lists-ruleset)
25. [Detailed DNS Settings Configuration Procedures](#25-detailed-dns-settings-configuration-procedures)
    * [Setting Up DNS Resolvers](#setting-up-dns-resolvers)
    * [Setting Up DNS Routing Rules](#setting-up-dns-routing-rules)
26. [Detailed Settings Configuration Procedures](#26-detailed-settings-configuration-procedures)
    * [Web Administration Panel Setup](#web-administration-panel-setup)
    * [Subscription Setup](#subscription-setup)
    * [Database Backups](#database-backups)

---

## 1. System Topology and Object Dependencies

The sing-box configuration follows a modular approach. All components reference each other using unique text labels (tags). Changing a parameter in one section often requires adjustments in dependent objects.

```
                    +------------------------------------+
                    |            DNS Settings            |
                    |    (Resolvers and routing rules)   |
                    +-----------------+------------------+
                                      |
                                      | queries through
                                      v
+------------------+        +------------------+        +------------------+
|     Inbounds     |        |      Rules       |        |    Outbounds     |
| (Listening ports,|------->|  (Routing policy |------->|  (Outbound proxy |
| protocols, TLS)  |        |    conditions)   |        |  & balance groups|
+--------+---------+        +------------------+        +--------+---------+
         |                                                       |
         | uses                                                  | restricted by
         v                                                       v
+------------------+                                    +------------------+
|   TLS Settings   |                                    |     Limiters     |
|  (Certificates / |                                    |  (Speed limits,  |
|     Reality)     |                                    |    max sessions) |
+------------------+                                    +------------------+
```

### Key Object References

| Parameter | References | Defined in | Purpose |
| :--- | :--- | :--- | :--- |
| **Inbound** (protocol) | TLS Profile | TLS Settings | Enables encryption, Reality, or classic TLS profiles |
| **Inbound** (protocol) | detour | Outbounds | Redirects decrypted streams (useful for ShadowTLS pairing) |
| **Outbound** (Failover, Selector) | outbounds | Outbounds (proxy tags) | Defines member proxies for balancing or routing |
| **Outbound** (proxy) | limiters | Outbounds (Limiter type) | Attaches bandwidth, traffic, or connection limits |
| **Rule** (Route) | inbound | Inbounds (listening tags) | Filters traffic by its incoming source port |
| **Rule** (Route) | outbound | Outbounds (outbound tags) | Directs matching traffic to the selected proxy or gateway |
| **DNS Server** | detour | Outbounds (proxy tags) | Sends DNS queries through the selected outbound path |

---

## 2. Inbounds (Incoming Connections)

Inbound connections open ports on the server to receive traffic from clients. Each entry becomes a sing-box listener with its own socket, authentication settings, and protocol stack.

### General Socket Settings (Listen Options)
Every inbound features a socket parameters block. Click the "Listen Options" button in the panel interface to show these hidden parameters.

*   **Listen:** The network IP address the service binds to.
    *   `0.0.0.0` binds to all available IPv4 interfaces (standard for public servers).
    *   `::` binds to all available IPv4 and IPv6 interfaces.
    *   `127.0.0.1` makes the port accessible only locally. This is used when deploying a listener behind a reverse proxy like Nginx, Apache, or Caddy.
*   **Port:** A number between 1 and 65535. Two services cannot share the same port.
*   **Detour:** The tag of another inbound to redirect decrypted traffic to (critical for ShadowTLS to Shadowsocks setups).
*   **Bind Interface:** The name of a physical or virtual network interface (e.g., `eth0` or `wg0`) to restrict listeners.
*   **Routing Mark:** A numeric mark in hex format (e.g., `0x2024`) applied to packets for system routing tables.
*   **Reuse Addr:** Allows the listener to bind to the port immediately after service restarts without waiting for the socket state to clear TIME_WAIT.
*   **Network Namespace (netns):** Binds the listening socket inside a virtual Linux network namespace for isolation.
*   **TCP Fast Open (TFO):** Enables data transmission inside the initial SYN packet. Reduces connection setup times.
*   **TCP Multi Path:** Enables simultaneous transmission over multiple network interfaces (e.g., Wi-Fi and mobile data).
*   **UDP Fragment:** Permits fragmentation of outgoing UDP packets when network MTU is low.
*   **UDP NAT Expiration:** Session lifetime for UDP translations in minutes. Closes idle sessions to save resources.
*   **Disable TCP Keep Alive:** Stops sending check packets during idle sessions.
*   **TCP Keep Alive:** The interval for sending check packets (e.g., `15s`).
*   **TCP Keep Alive Interval:** The time between retry checks when a keep-alive packet goes unacknowledged.

### Advanced Inbound Parameters
This block manages packet analysis and domain routing strategies.

*   **Sniff:** Inspects connection metadata to determine domain names. Necessary for domain-based routing.
*   **Sniff Override Destination:** Replaces the destination IP address of packets with the IP resolved from sniffed domains.
*   **Sniff Timeout:** The time limit for protocol analysis in milliseconds (e.g., `300ms`). Packets are passed through as raw data if analysis times out.
*   **Proxy Protocol:** Reads PROXY protocol headers (v1 or v2) to retrieve client IPs from frontend load balancers.
*   **Proxy Protocol Accept No Header:** Continues processing connections even when the PROXY header is missing.
*   **Domain Strategy:**
    *   `prefer_ipv4` queries both IPv4 and IPv6, preferring IPv4.
    *   `prefer_ipv6` prefers IPv6.
    *   `ipv4_only` restricts queries to IPv4.
    *   `ipv6_only` restricts queries to IPv6.
*   **UDP Disable Domain Unmapping:** Skips reverse resolution mapping for UDP packets, lowering CPU usage during gaming.

### Inbound Protocols
*   **VLESS:** A lightweight protocol without transport-layer encryption (delegates security to TLS or Reality).
    *   *Required fields:*
        *   `Decryption` must be set to `none`.
        *   `TLS Profile` (TLS or Reality profile).
*   **VMess:** The classic V2Ray protocol featuring custom encryption and UUID auth.
    *   *Specific settings:*
        *   `UUID` (Client identifier).
        *   `Alter ID` (Must be set to `0` in modern deployments).
        *   `Security` (Method: `auto`, `none`, `zero`, `aes-128-gcm`, `chacha20-poly1305`).
        *   `Packet Encoding` (`none`, `packetaddr`, or `xudp`).
        *   `Network` (TCP, gRPC, WebSocket).
        *   `Global Padding` (Appends noise to packets to mitigate DPI analysis).
        *   `Authenticated Length` (Encrypts packet length).
*   **Trojan:** Password-based proxy masquerading as HTTPS.
    *   *Specific settings:*
        *   `Fallback Server` (Redirects unauthorized users to a local web server, e.g., `127.0.0.1`).
        *   `Fallback Port` (The local web server port, e.g., `80` or `443`).
        *   `Fallback for ALPN` (Route mapping based on TLS ALPN headers).
        *   `TLS Profile` (Requires a valid certificate profile).
*   **Shadowsocks:** Secure proxy utilizing symmetric encryption.
    *   *Specific settings:*
        *   `Method` (Algorithm: `2022-blake3-aes-128-gcm` or `2022-blake3-chacha20-poly1305` recommended).
        *   `Password` (Pre-shared key. The panel generates valid keys automatically).
        *   `Network` (TCP, UDP, or both).
        *   `Managed` (Enables user accounting on this port).
*   **Socks:** Classic proxy without built-in transport encryption.
    *   *Specific settings:*
        *   `Username` & `Password` (Auth details).
        *   `Version` (`4`, `4a`, or `5`). Version `5` supports UDP and auth.
        *   `Network` (TCP, UDP).
        *   `UoT` (UDP over TCP encapsulation).
*   **Http:** HTTP proxy server. Not recommended for public use without TLS.
    *   *Specific settings:*
        *   `Username` & `Password` (Basic Auth).
        *   `Path` (Virtual proxy route path).
        *   `Headers` (Custom HTTP response headers).
*   **Mieru:** Stealth proxy protocol using behavioral obfuscation.
    *   *Specific settings:*
        *   `Transport` (Underlying transport mode).
        *   `User Hint is Mandatory` (Requires clients to submit user headers).
        *   `Listen Ports` (Array of ports handled by the listener).
        *   `Traffic Pattern` (Pattern description for noise generation).
*   **Sudoku:** Obfuscation protocol mimicking plain HTTP traffic.
    *   *Specific settings:*
        *   `Key` (Pre-shared key).
        *   `AEAD Method` (`chacha20-poly1305`, `aes-128-gcm`, or `none`).
        *   `Table Type` (Substitution matrix layout).
        *   `Padding Min` / `Padding Max` (Noise boundaries).
        *   `Handshake Timeout` (Authorization deadline).
        *   `Pure Downlink` (Server-to-client traffic tuning).
        *   `Custom Table` / `Custom Tables` (Custom substitution keys).
        *   `Disable Http Mask` (Turns off HTTP mimicry).
        *   `HTTP Mask Mode` (HTTP header profile).
        *   `Path Root` (Mock HTTP URL path).
        *   `Fallback` (Unauthorised redirect page).
*   **TrustTunnel:** Secure transport utilizing QUIC connection profiles.
    *   *Specific settings:*
        *   `Network` (TCP, UDP).
        *   `QUIC` (Enables UDP QUIC transport).
        *   `Congestion Controller` (Algorithm: `bbr`, `bbr_standard`, `bbr2`, `bbr2_variant`, `cubic`, `reno`).
        *   `BBR Profile` (Intensity: `standard`, `conservative`, `aggressive`).
        *   `CWND` (Max congestion window size).
*   **SSH:** Emulates an SSH server.
    *   *Specific settings:*
        *   `Server Version` (Version string).
        *   `Max Auth Tries` (Password attempt limit).
        *   `Host Key` / `Host Key Path` (Server private key values or file paths).
*   **TUIC:** Performance-oriented proxy running on QUIC.
    *   *Specific settings:*
        *   `Congestion Control` (`cubic`, `new_reno`, or `bbr`).
        *   `Zero-RTT Handshake` (Enables data transmission inside initial packets).
        *   `Auth Timeout` (Auth timeout in seconds).
        *   `Heartbeat` (NAT pinhole retention interval).
*   **MTProxy:** Telegram proxy featuring FakeTLS.
    *   *Specific settings:*
        *   `Prefer IP` (Selects IPv4 or IPv6 preference).
        *   `Concurrency` (Worker thread pools).
        *   `Auto Update` (Updates Telegram datacenter IP ranges).
        *   `Domain Fronting Port` (Masquerade port).
        *   `Fronting Host` (Masquerade target, e.g., `cloudflare.com`).
        *   `Proxy Protocol` (For frontend proxies).
        *   `Idle Timeout` / `Handshake Timeout` / `Tolerate Time Skewness`.
        *   `Throttle Max Connections` / `Throttle Check Interval`.
        *   `Allow Fallback on Unknown DC`.
        *   `Doppelganger URLs` (Mock TLS handshake websites).
        *   `Doppelganger Per Raid` / `Doppelganger Each` / `Doppelganger DRS`.
*   **Direct:** Relays raw traffic to target destinations.
    *   *Specific settings:*
        *   `Network` (TCP, UDP).
        *   `Override Address` & `Override Port` (Destination redirection).
*   **Tun:** Creates a system network interface card to capture OS-level traffic.
    *   *Specific settings:*
        *   `Addresses` (CIDR IP ranges, e.g., `172.18.0.1/30`).
        *   `Interface Name` (Defaults to `tun0`).
        *   `MTU` (Usually `1400` or `1500`).
        *   `UDP Timeout` (UDP session tracking limit).
        *   `Stack` (Slab network stack: `system`, `gvisor`, or `mixed`).
        *   `Auto Route` / `Auto Redirect` / `Strict Route` (Routing configurations).
        *   `Exclude MPTCP` / `Fallback Rule Index` / `Reset Mark`.
        *   `Input Mark` / `Output Mark` / `NFQueue`.
        *   `Keep Lan Direct` (Bypasses LAN subnets).
        *   `Rule Set Tunnel` (Routing list maps).
*   **TProxy:** Captures system traffic transparently without IP modifications.
    *   *Specific settings:*
        *   `Network` (TCP, UDP, or both).
*   **Bond Inbound:** Logical group multiplexing traffic from multiple listeners.
    *   *Specific settings:*
        *   `Inbounds` (Target inbound tags mapped together).
*   **Core Failover Inbound:** Provides native fallback capabilities for incoming listeners.
    *   *Specific settings:*
        *   `Inbounds` (Target inbound tags mapped for redundant listening).

---

## 3. Outbounds (Outgoing Connections)

Outbound profiles define destination proxies or direct routes.

*   **direct:** Resolves and contacts target servers directly from the host.
    *   `Override Address` & `Override Port` (Destination modifications).
*   **block:** Silently drops packets.
*   **dns:** Reroutes traffic to the built-in DNS engine.
*   **socks:** Outbound Socks4, Socks4a, or Socks5 client.
    *   `Server` / `Server Port` / `Username` / `Password` / `Version` / `UoT`.
*   **http:** Outbound HTTP client.
    *   `Server` / `Server Port` / `Username` / `Password` / `Headers` / `Path`.
*   **shadowsocks:** Outbound Shadowsocks client.
    *   `Method` / `Password` / `Plugin` / `Plugin Options`.
*   **vmess:** Outbound VMess client.
    *   `UUID` / `Alter ID` / `Security` / `Packet Encoding` / `Global Padding` / `Authenticated Length`.
*   **vless:** Outbound VLESS client.
    *   `UUID` / `Flow` / `Decryption` (Always set to `none`).
*   **trojan:** Outbound Trojan client.
    *   `Password` (Proxy password).
*   **naive:** Outbound NaiveProxy client.
    *   `Server` / `Server Port` / `Username` / `Password`.
*   **tor:** Routes traffic through Tor networks.
    *   `Tor RC Path` / `Tor Instance`.
*   **ssh:** Routes traffic through SSH tunnels.
    *   `User` / `Password` / `Private Key` / `Private Key Passphrase` / `Host Key`.
*   **shadowtls:** Outbound ShadowTLS client.
    *   `Password` / `Version`.
*   **anytls:** Outbound AnyTLS client.
    *   `Password` / `Padding Scheme`.
*   **mieru:** Outbound Mieru client.
    *   `Username` / `Password` / `Multiplexing` / `Server Ports` / `Traffic Pattern`.
*   **trusttunnel:** Outbound TrustTunnel client.
    *   `Username` / `Password` / `QUIC` / `Congestion Controller` / `BBR Profile`.
*   **sudoku:** Outbound Sudoku client.
    *   `Key` / `AEAD Method` / `Table Type` / `HTTP Mask` (enabled, mode, multiplex, host, path_root).
*   **masque:** Outbound MASQUE client on HTTP/3.
*   **openvpn:** Outbound OpenVPN client.
    *   `Config Path` (Path to `.ovpn` file).
*   **hysteria / hysteria2:** Outbound Hysteria client.
    *   `Obfs` (Obfuscation key) / `Upload / Download Mbps` (Speed limits for BBR calculations).
*   **tuic:** Outbound TUIC client.
    *   `UUID` / `Password` / `Congestion Control` / `Zero-RTT Handshake`.

---

## 4. Socket Connection Settings (Dial Options)

Dial Options determine how outgoing network sockets are initialized. Click the "Dial Options" button inside any Outbound config to reveal these configurations.

*   **Detour:** Redirects outgoing connections through another proxy (enables proxy chaining).
*   **Bind Interface:** Restricts sockets to a specific network interface.
*   **Inet4 Bind Address / Inet6 Bind Address:** Binds the socket to a specific local IPv4 or IPv6 address.
*   **Bind Address No Port:** Lets the OS choose local bind ports dynamically.
*   **Protect Path:** Android-specific path for bypassing VPN loops.
*   **Routing Mark:** Linux socket fwmark index.
*   **Reuse Addr:** Permits socket reuse.
*   **Network Namespace (netns):** Namespace binding.
*   **TCP Fast Open (TFO):** Enables TFO on outgoing connections.
*   **TCP Multi Path:** Multi-path transport configuration.
*   **Disable TCP Keep Alive:** Disables check packets.
*   **TCP Keep Alive / TCP Keep Alive Interval:** Check intervals.
*   **UDP Fragment:** Enables UDP fragmentation.
*   **Connect Timeout:** Timeout in seconds.
*   **Domain Resolver:** DNS server to resolve proxy addresses.
*   **Domain Strategy:** Resolution strategy.
*   **Fallback Delay:** Delayed IPv4 attempts during slow IPv6 responses.
*   **Fallback Network Type / Network Type:** Protocols.
*   **Network Strategy:** Multihoming strategies.
*   **Idle Session Timeout / Idle Session Check Interval / Min Idle Session.**

---

## 5. Endpoints (Network Tunnels)

Endpoints integrate virtual interfaces directly inside sing-box. This permits other VPN architectures to serve as outbound paths for user routing.

### WireGuard (Warp)
Integrates WireGuard (including Cloudflare Warp services) inside the proxy network stack.
*   **What it does:** Launches a virtual encrypted tunnel to a WireGuard host. Traffic routed into this endpoint exits the internet showing the IP of the remote WireGuard host.
*   **When to use:**
    *   To bypass regional blocks on websites without purchasing extra VPS nodes in multiple countries.
    *   To mask host server IPs when connecting to target servers.
    *   To route traffic to geo-blocked sites (e.g., media services) through Cloudflare Warp networks.
*   **Recommendations:** Add a rule in Rules mapping blocked target domains to this Endpoint. This allows clients to connect to geo-blocked sites without slowing down normal traffic.
*   **Parameters:**
    *   `Private Key`: Client private key.
    *   `Server / Server Port`: Remote endpoint address and port (for Cloudflare Warp, IP `162.252.172.57` and port `2408`).
    *   `Addresses`: Client virtual IP ranges separated by commas.
    *   `Reserved`: Custom client verification bytes.
    *   `MTU`: Interface packet size limit (usually `1280` or `1420`).

### Tailscale
Integrates the proxy host inside your Tailscale private mesh network.
*   **What it does:** Allows the host to connect securely to your home PCs, mobile devices, and other nodes inside a single private network without exposing administration ports to the public internet.
*   **When to use:**
    *   To secure host administration. You can bind the S-UI-X web dashboard port to a private Tailscale IP, preventing public scans.
    *   To coordinate backups and database connections across multiple remote nodes safely.
*   **Recommendations:** Use Tailscale in tandem with the integrated derp service to bypass strict client NAT firewalls.
*   **Parameters:**
    *   `Auth Key`: Authorization key generated in your Tailscale control panel.
    *   `Control URL`: Tailscale control plane URL (defaults to Headscale or Tailscale official servers).

### VPN
Integrates classic client-server VPN tunnels.
*   **What it does:** Relays sing-box traffic through external VPN servers.
*   **When to use:** When you need to forward specific client profiles through a corporate VPN tunnel.
*   **Parameters:** Client/Server VPN parameters.

---

## 6. Services (System Processes)

Services manage system-level tasks including local DNS orchestration, process memory tracking, performance profiling, and node synchronization.

### resolved
Integrates with systemd-resolved DNS daemons on Linux.
*   **What it does:** Translates DNS parameters configured inside sing-box straight to the host OS.
*   **When to use:** When running sing-box as an OS-level transparent proxy (TUN/TProxy). Prevents port 53 bind conflicts and stops host DNS leaks.
*   **Recommendations:** Consider enabling `resolved` when using Tun interfaces and host processes must follow the same DNS policy.
*   **Parameters:** No configuration parameters required.

### ssm-api
SUI telemetry endpoint service.
*   **What it does:** Exposes APIs to query CPU load, memory usage, and interface traffic.
*   **When to use:** To connect external monitoring systems (e.g., Prometheus, Grafana).

### derp
Runs an integrated Designated Encrypted Relay for Packets (DERP) node.
*   **What it does:** Acts as an encrypted relay server for Tailscale node traffic.
*   **When to use:** When two Tailscale nodes reside behind strict Symmetric NAT setups and cannot establish direct peer-to-peer tunnels. Traffic relays through this DERP node.
*   **Recommendations:** Deploy DERP on servers with high bandwidth capacity and low latency to ensure fast fallback tunnels for clients.
*   **Parameters:**
    *   `Port`: Service port.
    *   `STUN Port`: STUN query port (usually UDP `3478`).

### ocm / ccm
Cluster configuration synchronizers.
*   **What it does:** Syncs configs and client lists across multiple server nodes.
*   **When to use:** In cluster setups where a single panel coordinates configurations across multiple target nodes.

### oom-killer
Host memory safeguard.
*   **What it does:** Watches RAM consumption of the sing-box process. Restarts the service safely if threshold limits are exceeded.
*   **When to use:** To protect servers against crashes caused by memory leaks. Leaks can occur in heavy configurations with hundreds of client sessions on gRPC or WebSocket transports with Reality encryption.
*   **Recommendations:** Set memory limits to approximately 80 percent of total free VPS memory.
*   **Parameters:**
    *   `Memory Limit`: RAM capacity ceiling before safe restarts.

### profiler
Performance profiling debugger.
*   **What it does:** Exposes a local pprof debug interface to read CPU, RAM, and call stack profiles.
*   **When to use:** To trace CPU spikes or investigate memory leaks during core diagnostics. This service is debug-oriented and may be unavailable in standard release builds unless the binary was built with `with_profiler`.
*   **Recommendations:** Keep this service turned off in production. Turn it on temporarily only when troubleshooting system performance.
*   **Parameters:**
    *   `Port`: Local debug port.

---

## 7. TLS Settings

### Server TLS (Inbound Security)
*   **Certificate (CRT) & Private Key (KEY):** SSL key files.
*   **ALPN:** Protocol signatures (e.g., `h2`, `http/1.1`).
*   **Min Version / Max Version:** TLS versions (v1.2 and v1.3 recommended).
*   **Cipher Suites:** Cryptographic algorithms.

### Client TLS (Outbound Security)
*   **Server Name (SNI):** Host validation.
*   **Insecure:** Disables certificate checks (dangerous).
*   **Disable Session Resumption:** Prevents parameter reuse.

### Protocol Compatibility with TLS and Reality

When configuring security settings, understand which protocols support classic TLS (requires SSL certificates) and which are compatible with Reality technology (obfuscation using a trusted domain host).

#### Protocols Supporting Classic TLS
Classic TLS encrypts connections using a domain name and a valid certificate. Supported by:
*   **VLESS, VMess, Trojan:** Core proxy protocols using TCP, WebSocket, or gRPC transport networks.
*   **Hysteria 2 / TUIC:** UDP and QUIC based protocols. They require standard SSL certificates. Note: Reality does not work here because Reality requires TCP handshake interception.
*   **NaiveProxy:** Runs Caddy or Chromium stacks which require standard domain certificates.
*   **Socks / HTTP:** Sockets can be wrapped in TLS to secure links between clients and proxies.
*   **ShadowTLS, AnyTLS, TrustTunnel:** Use certificates to mimic or construct secure TLS paths.

#### Protocols Supporting Reality
Reality bypasses domain registrations and certificate management. It intercepts client handshakes and mocks them as TLS connections to large trusted websites. Supported by:
*   **VLESS:** The standard, most stable combination (VLESS + Reality + XTLS).
*   **Trojan:** sing-box supports Trojan in Reality mode.
*   **VMess:** VMess can utilize Reality over TCP transports.

*Reality limitations:* UDP/QUIC-based protocols (e.g., Hysteria, TUIC) are technically incompatible with Reality.

---

## 8. Rules and Rule Sets (Traffic Routing)

*   **Inbound:** Incoming listen tags.
*   **Client:** Profile username matches.
*   **Domain / Domain Suffix / Domain Keyword / Domain Regex:** Domain matches.
*   **IP / CIDR:** Destination subnets.
*   **Source IP / Source CIDR:** Client subnets.
*   **Port / Source Port:** Socket ports.
*   **Protocol:** Packet signatures (`http`, `tls`, `quic`, `dns`, `bittorrent`).
*   **Rule Sets:** Local or remote geo-rule lists.
*   **Outbound:** Route destination.

### Rule Parameter Fields
*   **Inbound:** Filters packets based on incoming listening tags.
*   **Client:** Associates rules to client usernames.
*   **Domain / Suffix / Keyword / Regex:** Evaluates requested web destinations.
*   **IP / CIDR:** Evaluates destination network subnets.
*   **Source IP / CIDR:** Evaluates client origin subnets.
*   **Port / Source Port:** Target and source socket ports.
*   **Protocol:** Identifies specific application protocols (HTTP, Bittorrent, etc.).
*   **Rule Sets:** Integrates external geo-rule databases.
*   **Outbound:** The destination outbound tag when filters match.

---

## 9. DNS Settings

### DNS Resolvers
*   **Address:** DNS targets. Supports IPs (`8.8.8.8`), DoT (`tls://1.1.1.1`), or DoH (`https://dns.google/dns-query`).
*   **Detour:** Redirects DNS requests through proxies to prevent eavesdropping.
*   **Client Subnet:** Passes EDNS0 headers.

### DNS Rules
*   **Domain / Domain Suffix:** Triggers.
*   **Server:** Target DNS resolver.
*   **Disable Cache:** Prevents caching.

---

## 10. Load Balancing and Fallbacks

### Selector
A manual choice interface exposed to client devices.
*   `tag` & `outbounds` (List of tags).

### URLTest
Selects the fastest server based on periodic latency checks.
*   `url` (Test URL), `interval`, `tolerance` (Prevents jumping when ping values fluctuate).

### Fallback
Switches connection pathways on socket dial timeouts. Sequential, zero latency ping requirement.
*   Ordered list of `outbounds`.

### Failover
Controlled by panel scripts. Active checks monitor availability.
*   `Hysteresis` (Pass requirement count to restore servers).
*   `All Down Policy` (`hold_current`, `block`, `direct` which bypasses proxies).

### Bond
Aggregates channels (modes: `balance-rr` round-robin, `active-backup`).

### CoreFailover
Native sing-box failover for dial-time member selection. In the panel this type is shown as `core-failover` and serialized to the core as native `type: failover`.

### Parser
Decodes external proxy subscription links into outbound configs.

---

## 11. Traffic Limiters

*   **BandwidthLimiter:** `speed` (Limit in bytes), `mode` (sum or separated limits).
*   **ConnectionLimiter:** `count` (Max TCP/UDP sockets).
*   **TrafficLimiter:** Blocks traffic above monthly quotas.
*   **RateLimiter:** PPS constraints.
*   **Block:** Explicit blacklist block.

---

## 12. Step-by-Step Scenarios

### Scenario 1: VLESS-Reality Setup (TLS camouflage)
1.  **TLS Configuration:**
    *   Navigate to TLS Settings and select add.
    *   Choose the Reality profile type.
    *   Specify target masquerade destination under `Dest` (e.g., `images.apple.com:443`) and SNI domain under `Server Names`.
    *   Generate secure public and private key pairs.
    *   Input a random hex value in Short IDs. Save the profile.
2.  **Inbound Configuration:**
    *   Navigate to Inbounds and select add.
    *   Choose type VLESS.
    *   Specify port `443` and attach the created Reality TLS profile.
    *   Enable Sniff and Sniff Override Destination. Save the settings.
3.  **Client Provisioning:**
    *   Navigate to Clients and add a user profile linked to the VLESS inbound.
    *   Copy the generated subscription URI or QR code.

### Scenario 2: Shadowsocks Setup (Classic Fast Proxy)
1.  **Inbound Configuration:**
    *   Navigate to Inbounds and select add.
    *   Choose type Shadowsocks.
    *   Select method `2022-blake3-aes-128-gcm`.
    *   Generate a secure password key. Save the settings.
2.  **Routing Configurations:**
    *   Create a rule in Rules forwarding this Shadowsocks traffic tag to direct or proxy detours.

### Scenario 3: ShadowTLS with Hidden Shadowsocks
1.  **Add local backing proxy:**
    *   Create a Shadowsocks Inbound locally on `127.0.0.1` on port `10001` (tag `ss-inner`).
2.  **Add external ShadowTLS gateway:**
    *   Create a ShadowTLS Inbound on external port `8443`.
    *   Set detour target to `ss-inner`.
    *   Select version `3`.
    *   Specify masquerade host (e.g., `cloudflare.com`). Save parameters.
3.  **Client setup:** Clients connect to port `8443`, complete TLS handshakes, and get automatically routed to Shadowsocks.

---

## 13. System Maintenance

### Subscription Security (Sub Secrets)
*   Unique client secrets (`subSecret`) block URL brute forcing.
*   Enabling `subSecretRequired` rejects legacy plain name links.
*   Subscription responses support gzip compression and utilize no-store caching headers.

### Client IP Monitoring
*   IP Hashing with private salts prevents raw IP database logs.
*   `Enforce` mode restricts devices beyond limits from opening new connections without tearing down existing sessions.

### Panel Updates and Database Backups
*   **Backups:** Database exports are streamed directly from SQLite on the disk to save host memory.
*   **Updates:** Checks file integrity using SHA-256 signatures, runs diagnostic starts in temporary directories, and rolls back binary installations automatically on failure.

---

## 14. Troubleshooting

*   `bind: address already in use`: Change ports or audit host sockets (`ss -tulpn`).
*   `permission denied` below 1024: Run cap config `setcap 'cap_net_bind_service=+ep' /usr/local/bin/s-ui` or launch as root.
*   DNS Leak: Map DNS detour rules properly, set Domain Strategy, and verify Sniffing is enabled.

---

## 15. Protocol Comparison Matrix

This comparison table helps you choose the correct proxy protocol depending on your network conditions and security goals.

| Protocol | DPI Censorship Bypass | Packet Loss Resilience | CPU Overhead on Host | Client Compatibility |
| :--- | :--- | :--- | :--- | :--- |
| **VLESS + Reality** | Excellent (TLS simulation) | Average (TCP bound) | Minimal (no proxy-level cipher) | Excellent (Nekobox, v2rayNG, Sing-box) |
| **Hysteria 2 / TUIC** | Excellent (UDP/3 mimicking) | Excellent (UDP BBR congestion) | High (intensive QUIC encryption) | Good (Nekobox, Sing-box, Shadowrocket) |
| **Shadowsocks 2022** | Average (signature blocks possible) | Average (TCP bound) | Minimal (optimized AES cipher) | Maximum (widely supported everywhere) |
| **ShadowTLS + SS** | Excellent (mimics domain handshake) | Average (TCP bound) | Average (double encapsulation cost) | Average (requires paired config setup) |
| **NaiveProxy** | Maximum (Chromium network stack) | Average (TCP bound) | High (Chromium engine emulation) | Average (requires plugin support) |
| **Socks / HTTP** | None (unobfuscated cleartext) | Low (large framing headers) | Zero | Maximum (OS native support) |

---

## 16. Glossary of Terms

*   **SNI (Server Name Indication):** An extension to the TLS handshake where the client sends the domain name it tries to connect to before encryption starts. DPI firewalls analyze SNI to drop blocked sites.
*   **ALPN (Application-Layer Protocol Negotiation):** A TLS extension that allows negotiated application protocols (e.g., HTTP/2, HTTP/1.1) to be declared during the secure handshake.
*   **CIDR (Classless Inter-Domain Routing):** A method for allocating IP addresses and routing IP packets. A CIDR record like `192.168.1.0/24` declares a subnet using mask `255.255.255.0`.
*   **BBR (Bottleneck Bandwidth and RTT):** Google-engineered congestion control algorithm. It models network bandwidth and packet round-trip delays instead of relying solely on packet drops.
*   **FWMARK (Flow Mark):** A mark set on packets inside the Linux kernel to apply advanced routing rules.
*   **Netns (Network Namespace):** A Linux kernel feature providing isolated network environments (interfaces, routes, rules).
*   **EDNS0 (Extension Mechanisms for DNS):** A DNS extension enabling clients to attach subnet information to direct DNS servers to serve geographically close CDN nodes.
*   **DPI (Deep Packet Inspection):** Advanced packet filtering checking data payloads to identify and block proxy protocol signatures.

---

## 17. Security and Testing Checklist

Follow these steps to confirm your proxy server remains secure and operating correctly.

1.  **IP Leak Test:**
    *   Navigate to an IP lookup tool (e.g., `ipleak.net` or `whoer.net`) through the configured proxy.
    *   Confirm only your proxy server IP is shown, and no residential or cellular client IP leaks out.
2.  **DNS Leak Test:**
    *   Run an extended DNS leak test (e.g., on `dnsleaktest.com`).
    *   Only the DNS servers specified in your S-UI-X DNS config must show up. No DNS servers belonging to your home or mobile ISP should be visible.
3.  **Securing Panel Administration:**
    *   Enforce HTTPS encryption for the administration panel (Security configuration in Settings).
    *   Set the `subSecretRequired` option to true to safeguard subscription paths against enumeration attacks.
4.  **IP Limit Enforcement Check:**
    *   Configure a client with an IP limit of 2 devices. Try connecting with 3 devices concurrently.
    *   The third device must fail to connect while the first two devices remain connected and stable.

---

## 18. Differences Between Similar Network Mechanisms

### CoreFailover vs Failover (Panel-Managed)
Both mechanisms can move new connection attempts to backup members, but they do it differently:
*   **CoreFailover (native core):** A sing-box handler that chooses members at dial time. When a dial attempt fails, the core can try the next configured member. The panel stores this as `core-failover` and expands tag references into native core objects during config assembly.
*   **Failover (panel-managed):** A panel-level priority controller. The panel checks members periodically over HTTP, applies hysteresis, and updates the active member for new sessions. Existing sessions can still break when the active member changes.

### Bond vs Selector / URLTest
*   **Bond (Aggregation):** Combines physical proxy channels into a single virtual interface. Running in round-robin mode, it spreads packet loads across all servers concurrently to accelerate multi-threaded file transfers.
*   **Selector (Manual Routing):** Does not load balance traffic. Instead, it routes packets strictly through a specific proxy tag selected manually by the user or administrator.
*   **URLTest (Latency-Based Routing):** Regularly probes proxy latency and automatically directs all traffic to the server with the lowest ping response. Unlike Bond, traffic goes through only one active path.

---

## 19. Differences Between Inbound and Outbound Configurations

Several protocols exist as both Inbound and Outbound targets, but their usage differs significantly:

*   **Socks / HTTP:**
    *   *Inbound:* Opens a local port on your host, allowing web browsers or external tools to proxy traffic through the server. Requires local bind parameters (e.g., `127.0.0.1` or `0.0.0.0`) and a port.
    *   *Outbound:* Directs sing-box to route traffic through an external Socks or HTTP proxy. Requires host target addresses and ports.
*   **SSH:**
    *   *Inbound:* Runs an integrated SSH server on a port. Clients connect using SSH commands to forward ports.
    *   *Outbound:* Directs sing-box to connect to an external SSH host using SFTP/SSH protocols to route proxy traffic.
*   **Direct:**
    *   *Inbound:* Listens on a port and forwards incoming cleartext traffic to target destinations (port forwarding).
    *   *Outbound:* Final routing target. Forwards packets straight to the internet using the host interface cards.

---

## 20. Detailed Protocol Configuration Procedures

Detailed steps to create Inbound and Outbound services for each protocol including value formats:

### Inbound Configurations

#### VLESS
1.  **TLS Setup:** Navigate to TLS Settings and create a TLS (classic VLESS) or Reality (obfuscated VLESS) profile.
2.  **Inbound Configurations:**
    *   `Type`: Select `vless`.
    *   `Port`: Input an unused port (e.g., `443` is standard for Reality).
    *   `Decryption`: Input `none`.
    *   `TLS`: Select your created TLS/Reality profile.
3.  **User Binding:** Add a client user under the `Clients` tab. UUIDs generate automatically.

### VMess
1.  **Inbound Configurations:**
    *   `Type`: Select `vmess`.
    *   `Port`: Input an unused port (e.g., `10002`).
    *   `UUID`: Input a valid UUIDv4 string (e.g., `f81d4fae-7dec-11d0-a765-00a0c91e6bf6`). Use the generate button next to the input.
    *   `Security`: Select `auto`.
2.  **Transport Network:** Choose WebSocket or gRPC if needed. Input a path in the `Path` field (e.g., `/ws`).
3.  **TLS:** Select a TLS profile if transport-level encryption is required.

### Trojan
1.  **Inbound Configurations:**
    *   `Type`: Select `trojan`.
    *   `Port`: Input a port (usually `443`).
    *   `Fallback Server`: Input a local web server target (e.g., `127.0.0.1`).
    *   `Fallback Port`: Input the local web port (e.g., `80` or `8080`).
2.  **TLS:** Select a standard TLS certificate profile (Trojan does not run without encryption).

### Shadowsocks
1.  **Inbound Configurations:**
    *   `Type`: Select `shadowsocks`.
    *   `Port`: Input a port (e.g., `10003`).
    *   `Method`: Select `2022-blake3-aes-128-gcm` or `2022-blake3-chacha20-poly1305`.
    *   `Password`: Input a pre-shared key. 2022 ciphers require keys of specific lengths (16 or 32 bytes in base64). Use the generate key button.
    *   `Network`: Select `tcp` or `udp` (both recommended).

### Socks / HTTP
1.  **Inbound Configurations:**
    *   `Type`: Select `socks` or `http`.
    *   `Port`: Input a port (e.g., `1080` for Socks, `8080` for HTTP).
    *   `Username` & `Password`: Input auth details. Leave empty to permit anonymous access.
    *   `UoT` (Socks): Enable if client UDP traffic needs routing over TCP.

### Hysteria 2
1.  **Inbound Configurations:**
    *   `Type`: Select `hysteria2`.
    *   `Port`: Input a UDP port (e.g., `443` or `20000`).
    *   `Obfs`: Input an obfuscation key in the `password` field if QUIC header hiding is desired.
2.  **TLS:** Select a TLS profile (Reality is not supported by Hysteria).
3.  **Bandwidth:** Input upload and download limits in Mbps (e.g., `100`).

### TUIC
1.  **Inbound Configurations:**
    *   `Type`: Select `tuic`.
    *   `Port`: Input a UDP port.
    *   `Congestion Control`: Select BBR congestion control.
    *   `Auth Timeout`: Input an auth timeout (e.g., `3s`).
2.  **TLS:** Select a standard TLS profile.

### NaiveProxy
1.  **Inbound Configurations:**
    *   `Type`: Select `naive`.
    *   `Port`: Input a port.
    *   `Username` & `Password`: Input auth details.
2.  **TLS:** Select a standard TLS profile.

### ShadowTLS
1.  **Add Backing Proxy:** Create a Shadowsocks Inbound locally on `127.0.0.1` port `10001` (tag `ss-inner`).
2.  **Inbound Configurations:**
    *   `Type`: Select `shadowtls`.
    *   `Port`: Input an external port (e.g., `8443`).
    *   `Detour`: Select `ss-inner`.
    *   `Version`: Select version `3`.
    *   `TLS`: Input a masquerade domain (e.g., `cloudflare.com`).

### Mieru
1.  **Inbound Configurations:**
    *   `Type`: Select `mieru`.
    *   `Transport`: Choose transport mode.
    *   `Listen Ports`: Input listening ports array.
    *   `Traffic Pattern`: Input path to the traffic pattern file.

### Sudoku
1.  **Inbound Configurations:**
    *   `Type`: Select `sudoku`.
    *   `Key`: Input a pre-shared key.
    *   `AEAD Method`: Select `chacha20-poly1305` or `aes-128-gcm`.
    *   `Table Type`: Select matrix table type.

### TrustTunnel
1.  **Inbound Configurations:**
    *   `Type`: Select `trusttunnel`.
    *   `Network`: Choose network protocols.
    *   `QUIC`: Enable QUIC over UDP.
    *   `Congestion Controller`: Select congestion control (e.g., `bbr`).

### SSH
1.  **Inbound Configurations:**
    *   `Type`: Select `ssh`.
    *   `Port`: Input a port (e.g., `2222`).
    *   `Host Key Path`: Input the host private key path (e.g., `/etc/ssh/ssh_host_rsa_key`).

### MTProxy
1.  **Inbound Configurations:**
    *   `Type`: Select `mtproxy`.
    *   `Port`: Input a port.
    *   `Fronting Host`: Input fronting target domain (e.g., `cloudflare.com`).
    *   `Concurrency`: Input thread count (e.g., `2`).

### Tun
1.  **Inbound Configurations:**
    *   `Type`: Select `tun`.
    *   `Addresses`: Input CIDR ranges (e.g., `172.19.0.1/30`).
    *   `Interface Name`: Input interface name (e.g., `tun0`).
    *   `Stack`: Select network stack (`gvisor` recommended).
    *   `Auto Route`: Enable.

### Redirect / TProxy
1.  **Inbound Configurations:**
    *   `Type`: Select `redirect` or `tproxy`.
    *   `Port`: Input firewall redirection port (e.g., `10080` for Redirect, `10081` for TProxy).
    *   `Network`: Select `tcp` for Redirect, `tcp,udp` for TProxy.

### Outbound Configurations

Detailed steps to create Outbound connections for each protocol:

#### direct (Direct Route)
1.  **Outbound Setup:**
    *   `Type`: Select `direct`.
    *   `Override Address`: Input an IP or domain target if you need to redirect all traffic going through this outbound to a specific host (leave blank for standard direct routing).
    *   `Override Port`: Input a port number (leave blank or `0` to preserve the original destination port).

#### block (Blackhole block)
1.  **Outbound Setup:**
    *   `Type`: Select `block`.
    *   No parameters needed. This handler is used in Rules to blackhole unwanted traffic (e.g., ad domains).

#### socks (SOCKS Client)
1.  **Outbound Setup:**
    *   `Type`: Select `socks`.
    *   `Server`: Input the remote SOCKS server IP or domain (e.g., `192.168.1.50` or `proxy.example.com`).
    *   `Port`: Input the remote proxy port.
    *   `Version`: Select `4`, `4a`, or `5`.
    *   `Username` & `Password`: Input auth details. Leave blank if the remote proxy does not require credentials.

#### http (HTTP Client)
1.  **Outbound Setup:**
    *   `Type`: Select `http`.
    *   `Server`: Input the remote HTTP proxy IP or domain.
    *   `Port`: Input the remote proxy port.
    *   `Username` & `Password`: Input auth details if required.
    *   `Path`: Input the virtual proxy path.

#### shadowsocks (Shadowsocks Client)
1.  **Outbound Setup:**
    *   `Type`: Select `shadowsocks`.
    *   `Server` & `Port`: Input remote Shadowsocks credentials.
    *   `Method`: Select the cipher method matching the remote server (e.g., `2022-blake3-aes-128-gcm`).
    *   `Password`: Input the remote pre-shared key.

#### vmess (VMess Client)
1.  **Outbound Setup:**
    *   `Type`: Select `vmess`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `UUID`: Input your account UUID.
    *   `Security`: Choose cipher (e.g., `auto`).

#### vless (VLESS Client)
1.  **Outbound Setup:**
    *   `Type`: Select `vless`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `UUID`: Input your account UUID.
    *   `Flow`: Input `xtls-r-ux-y` if flow control is enabled on the remote server (leave blank for standard VLESS).
    *   `Decryption`: Input `none`.

#### trojan (Trojan Client)
1.  **Outbound Setup:**
    *   `Type`: Select `trojan`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Password`: Input proxy password.

#### naive (NaiveProxy Client)
1.  **Outbound Setup:**
    *   `Type`: Select `naive`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Username` & `Password`: Input auth details.

#### tor (Tor Client)
1.  **Outbound Setup:**
    *   `Type`: Select `tor`.
    *   `Tor RC Path`: Input your Tor configuration file path (e.g., `/etc/tor/torrc`).

#### ssh (SSH Client)
1.  **Outbound Setup:**
    *   `Type`: Select `ssh`.
    *   `Server` & `Port`: Input remote SSH host parameters (usually port `22`).
    *   `User` & `Password`: Input auth credentials.
    *   `Private Key`: Paste your private key string if key-based authentication is configured.

#### shadowtls (ShadowTLS Client)
1.  **Outbound Setup:**
    *   `Type`: Select `shadowtls`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Password`: Input the pre-shared secret.
    *   `Version`: Select protocol version (e.g., `3`).

#### anytls (AnyTLS Client)
1.  **Outbound Setup:**
    *   `Type`: Select `anytls`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Password`: Input auth credentials.

#### Mieru (Mieru Client)
1.  **Outbound Setup:**
    *   `Type`: Select `mieru`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Username` & `Password`: Input auth credentials.
    *   `Multiplexing`: Select multiplex mode.

#### trusttunnel (TrustTunnel Client)
1.  **Outbound Setup:**
    *   `Type`: Select `trusttunnel`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Username` & `Password`: Input auth credentials.
    *   `QUIC`: Enable if using QUIC.

#### sudoku (Sudoku Client)
1.  **Outbound Setup:**
    *   `Type`: Select `sudoku`.
    *   `Server` & `Port`: Input remote server endpoints.
    *   `Key`: Input the pre-shared secret.
    *   `AEAD Method`: Select matching cipher.

#### masque (Masque Client)
1.  **Outbound Setup:**
    *   `Type`: Select `masque`.
    *   `Server` & `Port`: Input remote HTTP/3 MASQUE endpoints.

#### openvpn (OpenVPN Client)
1.  **Outbound Setup:**
    *   `Type`: Select `openvpn`.
    *   `Config Path`: Input local path to the `.ovpn` configuration file.

#### hysteria / hysteria2 (Hysteria Client)
1.  **Outbound Setup:**
    *   `Type`: Select `hysteria` or `hysteria2`.
    *   `Server` & `Port`: Input remote UDP endpoints.
    *   `Obfs`: Input password key in the `password` field if obfs is required.
    *   `Upload / Download Mbps`: Input local bandwidth limits in Mbps.

#### tuic (TUIC Client)
1.  **Outbound Setup:**
    *   `Type`: Select `tuic`.
    *   `Server` & `Port`: Input remote UDP endpoints.
    *   `UUID` & `Password`: Input auth credentials.
    *   `Congestion Control`: Select congestion control algorithms matching the server (e.g., `bbr`).

---

## 21. Detailed Endpoint Configuration Procedures

Endpoints integrate virtual interfaces directly inside sing-box.

### WireGuard (Warp)
1.  Add a new endpoint with type `wireguard`.
2.  Input configuration parameters:
    *   `Private Key`: Input your WireGuard client private key in base64 format.
    *   `Server` & `Server Port`: Input remote endpoint parameters (for Cloudflare Warp, IP `162.252.172.57` and port `2408`).
    *   `Addresses`: Input client IP ranges (e.g., `172.16.0.2/32`, `2001:db8::2/128`).
    *   `MTU`: Input interface packet size limit (e.g., `1280` or `1420`).
    *   `Reserved` (bytes array): Input custom client verification bytes if required by Warp.

### Tailscale
1.  Add a new endpoint with type `tailscale`.
2.  Input configuration parameters:
    *   `Auth Key`: Input the node authorization key generated in your Tailscale control panel.
    *   `Control URL`: Input your Tailscale control plane target URL (leave blank to default to `https://controlplane.tailscale.com`).

### VPN
1.  Add a new endpoint with type `vpn`.
2.  Specify network interfaces and key exchange parameters matching your VPN provider specifications.

---

## 22. Detailed Service Configuration Procedures

Integrated Services expand core sing-box features.

### resolved
1.  Add a new service with type `resolved`.
2.  No parameters are required. The service automatically integrates sing-box DNS configuration with systemd-resolved on Linux OS.

### ssm-api
1.  Add a new service with type `ssm-api`.
2.  Enables SUI telemetry APIs. Set listen IP and port values if customized.

### derp
1.  Add a new service with type `derp`.
2.  Input configuration parameters:
    *   `Port`: Input incoming requests TCP port.
    *   `STUN Port`: Input UDP STUN port (defaults to `3478`).

### ccm / ocm
1.  Add a new service with type `ccm` or `ocm`.
2.  Input server URLs and authorization tokens to synchronize configurations across nodes.

### oom-killer
1.  Add a new service with type `oom-killer`.
2.  Input configuration parameters:
    *   `Memory Limit`: Input maximum memory capacity limit in bytes (e.g., `536870912` for 512 MB). The watcher restarts the sing-box core safely if thresholds are crossed.

### profiler
1.  Add a new service with type `profiler`.
2.  Input configuration parameters:
    *   `Port`: Input local debug profiling port (e.g., `6060`).

---

## 23. Detailed TLS Settings Configuration Procedures

TLS Settings manage transport encryption profiles.

### Creating Server TLS Profiles (For Inbounds)
1.  Navigate to TLS Settings and select add.
2.  Input configuration parameters:
    *   `Name`: Input a unique profile label (e.g., `my-cert`).
    *   `Certificate (CRT)`: Paste the public SSL certificate chain.
    *   `Private Key (KEY)`: Paste the matching private SSL key.
    *   `ALPN`: Input accepted application protocol headers (e.g., `h2,http/1.1`).
    *   `Min Version` & `Max Version`: Select TLS versions (v1.2 and v1.3 recommended).

### Creating Reality Profiles
1.  Navigate to TLS Settings and select Reality.
2.  Input configuration parameters:
    *   `Dest`: Input target website domain and port (e.g., `images.apple.com:443`).
    *   `Server Names`: Input target masquerade domain list separated by commas.
    *   `Private Key` & `Public Key`: Generate keys using the generate button.
    *   `Short IDs`: Add at least one random hex identifier (2 to 16 characters).

---

## 24. Detailed Rules and Rulesets Configuration Procedures

Rules and Rulesets configure destination route conditions.

### Creating Routing Rules (Rule)
1.  Navigate to Rules and select add.
2.  Select packet routing conditions:
    *   `Inbound`: Choose listen tags to map incoming connections under this rule.
    *   `Client`: Select client usernames for custom user-based routing.
    *   `Domain Suffix`: Specify targeted domain extensions (e.g., `de` to route German sites directly).
    *   `IP / CIDR`: Specify target IP addresses or subnets.
    *   `Protocol`: Choose connection signatures (e.g., `bittorrent` to block torrent connections).
    *   `Rule Sets`: Choose local or remote rule databases.
3.  `Outbound`: Select destination outbound tag (e.g., `direct` or `block`).

### Creating Rule Lists (Ruleset)
1.  Navigate to Rulesets and select add.
2.  Select source profile type:
    *   `local`: Specify local path to `.srs` or `.json` database file on the server.
    *   `remote`: Input remote rule database URL and set update check intervals.

---

## 25. Detailed DNS Settings Configuration Procedures

DNS Resolvers convert hostnames into IP targets.

### Setting Up DNS Resolvers
1.  Navigate to DNS and select add server.
2.  Input configuration parameters:
    *   `Address`: Input DNS target address. Supports IPs (`8.8.8.8`), DoT (`tls://1.1.1.1`), or DoH (`https://dns.google/dns-query`).
    *   `Detour`: Select an outbound proxy tag to route DNS requests securely.
    *   `Client Subnet`: Input customer subnets to get optimized local CDN responses.

### Setting Up DNS Routing Rules
1.  Navigate to DNS Rules and select add.
2.  Configure domain triggers (e.g., domain suffix `de`).
3.  `Server`: Select a target DNS server to handle matching domains.

---

## 26. Detailed Settings Configuration Procedures

Settings configure panel-wide defaults.

### Web Administration Panel Setup
1.  `webListen`: Input panel access bind IP. For security, binding to `127.0.0.1` and deploying Nginx as an SSL reverse proxy is recommended.
2.  `webPort`: Input panel port (defaults to `2095`).
3.  `webPath`: Input secret url path prefix (e.g., `/secret-admin-sui/`) to hide the login portal from bots.

### Subscription Setup
1.  `subSecretRequired`: Enforce unique client token secrets inside proxy links.
2.  `subRateLimitPerIP`: Input maximum subscription requests per client IP per minute to protect system resources.

### Database Backups
1.  `DB Backup Schedule`: Configure backup cron parameters.
2.  Input your Telegram bot API key and chat ID parameters to deliver encrypted backup files automatically.
