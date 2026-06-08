<div align="center">

```
██████╗  █████╗ ██╗    ██╗██╗   ██╗███████╗██╗      ██████╗
██╔══██╗██╔══██╗██║    ██║██║   ██║██╔════╝██║     ██╔═══██╗
██████╔╝███████║██║ █╗ ██║██║   ██║█████╗  ██║     ██║   ██║
██╔══██╗██╔══██║██║███╗██║╚██╗ ██╔╝██╔══╝  ██║     ██║   ██║
██║  ██║██║  ██║╚███╔███╔╝ ╚████╔╝ ███████╗███████╗╚██████╔╝
╚═╝  ╚═╝╚═╝  ╚═╝ ╚══╝╚══╝   ╚═══╝  ╚══════╝╚══════╝ ╚═════╝
```

### The kernel stack doesn't know we exist.

[![Release](https://img.shields.io/github/v/release/justinahero/RawVelo-core?style=flat-square&color=ff4444)](https://github.com/justinahero/RawVelo-core/releases)
[![License](https://img.shields.io/github/license/justinahero/RawVelo-core?style=flat-square&color=444)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square)](https://go.dev)
[![Stars](https://img.shields.io/github/stars/justinahero/RawVelo-core?style=flat-square&color=gold)](https://github.com/justinahero/RawVelo-core/stargazers)

</div>

---

## What is RawVelo?

RawVelo is a **raw-socket KCP tunnel** that operates entirely beneath the OS network stack.

It uses **libpcap** to capture and inject packets directly at the NIC level — completely invisible to `conntrack`, `iptables` state tracking, and any kernel-level inspection. The tunnel doesn't create sockets the OS can see. It doesn't show up in `netstat`. It doesn't leave traces in `/proc/net`.

On top of that, it runs **KCP** — a reliable protocol over raw TCP frames — giving you low-latency, high-throughput tunneling that survives lossy and restricted networks.

```
[ Your App ]
     │
     ▼
[ RawVelo Client ]  ──── raw TCP frames (KCP) ────►  [ RawVelo Server ]
  pcap capture                                          pcap inject
  XOR obfuscation                                       XOR deobfuscation
  TLS camouflage                                        TLS camouflage
  random padding                                        strip padding
     │                                                       │
     ▼                                                       ▼
[ NIC Layer ]                                         [ NIC Layer ]
```

---

## Why RawVelo?

| Feature | Regular VPN | RawVelo |
|---------|-------------|---------|
| Kernel socket exposure | ✅ visible | ❌ invisible |
| conntrack entries | ✅ yes | ❌ none |
| DPI fingerprint | ✅ detectable | ❌ obfuscated |
| Survives RST injection | ❌ no | ✅ yes |
| Adaptive FEC | ❌ no | ✅ yes |
| TLS camouflage | ❌ no | ✅ yes |

---

## Features

- **Raw Socket Engine** — pcap-level packet capture and injection, zero kernel socket exposure
- **KCP Transport** — reliable, low-latency protocol over raw TCP frames
- **XOR Obfuscation** — rotating key scrambling breaks static DPI signatures
- **Random Padding** — variable packet sizes destroy size-based fingerprinting
- **Timing Jitter** — random inter-packet delays break timing analysis
- **TLS Camouflage** — handshake mimics real TLS 1.3, traffic looks like HTTPS
- **Adaptive FEC** — forward error correction auto-tunes every 5s based on real link loss
- **Least-connections LB** — smart load balancing across parallel connections
- **Exponential Backoff** — auto-reconnect with 2s → 60s backoff on connection loss
- **SOCKS5 Proxy** — drop-in proxy for any application
- **TCP/UDP Forwarding** — direct port forwarding mode

---

## Installation

```bash
curl -L https://github.com/justinahero/RawVelo-core/releases/latest/download/rawvelo-linux-amd64 \
  -o /usr/local/bin/rawvelo && chmod +x /usr/local/bin/rawvelo
```

---

## Quick Setup

### Step 1 — Generate keys
```bash
rawvelo secret   # → your encryption key
rawvelo secret   # → your obfs key (use a different one)
```

### Step 2 — Server: apply firewall rules
```bash
PORT=8443

# bypass conntrack — critical for raw socket to work
iptables -t raw    -A PREROUTING -p tcp --dport $PORT -j NOTRACK
iptables -t raw    -A OUTPUT     -p tcp --sport $PORT -j NOTRACK
iptables -t mangle -A OUTPUT     -p tcp --sport $PORT --tcp-flags RST RST -j DROP
iptables -I INPUT  -p tcp --dport $PORT -j ACCEPT
```

### Step 3 — Configure
```bash
# Server
rawvelo run -c /etc/rawvelo/server.yaml

# Client
rawvelo run -c /etc/rawvelo/client.yaml
# SOCKS5 now available at 127.0.0.1:1080
```

See [`example/`](example/) for full config reference.

---

## Configuration

### KCP Modes

| Mode | Interval | Best For |
|------|----------|----------|
| `normal` | 20ms | Stable, low-loss links |
| `fast` | 10ms | General use |
| `fast2` | 8ms | Low latency |
| `fast3` | 5ms | **Default** — minimum latency |
| `extreme` | 2ms | Best quality links only |
| `manual` | custom | Full parameter control |

### Obfuscation

```yaml
obfs:
  enabled: true
  key: "your-obfs-key"       # separate from encryption key
  padding: true              # randomize packet sizes
  jitter: false              # add timing delays (increases latency slightly)
  camouflage: false          # mimic TLS 1.3 (for heavily censored networks)
  adaptive_fec: true         # auto-tune FEC based on link loss
```

---

## Commands

```
rawvelo run      Run tunnel from config file
rawvelo secret   Generate a random encryption/obfs key
rawvelo ping     Test connectivity to server
rawvelo dump     Capture and display raw packets (debug)
rawvelo iface    List available network interfaces
rawvelo version  Print version and commit hash
```

---

## Building from Source

```bash
git clone https://github.com/justinahero/RawVelo-core
cd RawVelo-core
go build -o rawvelo ./cmd
```

Requires `libpcap-dev` on Linux.

---

## Management Script

For a full interactive deployment UI with kernel tuning, tunnel management, and health monitoring, see the **[RawVelo Paqet](https://github.com/justinahero/RawVelo)** management script.

---

## Support the Project

If RawVelo has been useful to you, consider supporting development:

**TRON (TRX / USDT-TRC20)**
```
TBMenySPuZYyre5S5imWJbvXXHSrXZcE3x
```

**TON**
```
UQDfANFVOcM_vsVXGgXmEYcMJvbGklcOHyTCtGguOjMn0QsL
```

---

## License

MIT © 2026 [justinahero](https://github.com/justinahero)

> *Built for the edge cases. For the networks that fight back.*
