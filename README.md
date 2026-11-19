<div align="center">
  <img width="269" height="188" alt="MESHadb" src="https://github.com/user-attachments/assets/03d0b879-ca5e-4a20-af72-533ea3ae2b52" />

  <p>
    <a href="https://docs.meshforensics.org/"><img src="https://img.shields.io/badge/docs-latest-blue.svg?style=flat-square" alt="Documentation" /></a>
    <a href="#"><img src="https://img.shields.io/badge/status-public%20beta-blueviolet?style=flat-square" alt="Public Beta Status" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/BARGHEST-ngo/MESH?style=flat-square" alt="License" /></a>
    <a href="https://github.com/BARGHEST-ngo/MESH/releases"><img src="https://img.shields.io/github/v/release/BARGHEST-ngo/MESH?include_prereleases&style=flat-square" alt="Latest Release" /></a>
  </p>
  <p>
    <a href="https://play.google.com/store/apps/details?id=com.barghest.mesh"><img height="60" alt="Get it on Google Play" src="https://play.google.com/intl/en_us/badges/static/images/badges/en_badge_web_generic.png" /></a>
    <a href="https://f-droid.org/en/packages/com.barghest.mesh/"><img height="60" alt="Get it on F-Droid" src="https://fdroid.gitlab.io/artwork/badge/get-it-on.png" /></a>
  </p>
  <p><a href="https://docs.meshforensics.org/">DOCS</a> &nbsp;|&nbsp; <a href="https://github.com/BARGHEST-ngo/MESHnet">MESHnet</a> &nbsp;|&nbsp; <a href="https://Barghest.asia">BARGHEST</a></p>
</div>

# MESHadb

**Peer-to-peer ADB over the internet. Reach an Android device's wireless debugging from anywhere, with no port forwarding, no accounts and no server to run.**

ADB wireless debugging only works when your computer and the phone share a Wi-Fi network. MESHadb removes that limit. The phone runs a small app, your computer runs a CLI, and the two join an encrypted tunnel using an address and a one-time PIN. From then on, `adb` talks to the phone as if it were on your desk.

- **Peer-to-peer first.** Your computer and the phone connect directly to each other over WireGuard-encrypted UDP, punching through NAT on both sides. There is no VPN hub in the middle.
- **HTTPS relay when direct fails.** On networks that block UDP or defeat hole punching, traffic falls back to a relay over HTTPS (TCP port 443), which looks like ordinary web traffic. The relay only forwards packets that are already end-to-end encrypted, so it cannot read them.

Sessions are ephemeral. Nothing is persisted, every start gets a new address, and when the session ends the tunnel is gone.

## What you can do with it

- **Remote support**: help a colleague or relative with their phone without asking them to touch router settings.
- **Development and QA**: debug an app on a device you do not physically hold.
- **Field devices**: reach phones and tablets deployed behind NAT or carrier networks.
- **Mobile forensics**: run a consensual remote acquisition, ready for [WARD](https://github.com/BARGHEST-ngo/WARD) or [MVT](https://github.com/mvt-project/mvt).
- **Exit node**: send a device's internet traffic out through your machine, to capture and inspect it or to get around a censored network.

## Quick start

You need:

- [Docker](https://docs.docker.com/get-docker/) and [Task](https://taskfile.dev/installation/) on your computer
- An Android phone with Wireless debugging (Android 11 or later)
- The MESHadb app, from [Google Play](https://play.google.com/store/apps/details?id=com.barghest.mesh), [F-Droid](https://f-droid.org/en/packages/com.barghest.mesh/) or the [releases page](https://github.com/BARGHEST-ngo/MESH/releases)

**1. Start the CLI**

```
git clone https://github.com/BARGHEST-ngo/MESH.git
cd MESH
task analyst
```

It prints a QR code and a MESH address, then drops you at the `meshcli>` prompt.

**2. Create a PIN**

```
meshcli> pin
PIN: 123456
```

**3. Join from the phone**

Open the MESHadb app, scan the QR code and enter the PIN.

**4. Pair ADB**

```
meshcli> adbpair
```

On the phone, turn on Wireless debugging and tap *Pair device with pairing code*. Type the code and the two ports into the prompt when asked.

**5. Use ADB**

Open a second terminal:

```
task shell
adb devices
adb shell
```

When you are done, run `disconnect` in the CLI, or `task down` to stop everything.

## Commands

| Command | What it does |
| --- | --- |
| `pin` | Create a one-time PIN so a device can join |
| `adbpair` | Pair and connect to a device's wireless debugging |
| `status` | Show the session and its connected devices |
| `forward on\|off` | Send a device's internet traffic through your machine |
| `capture start\stop` | Record a device's forwarded traffic to a pcap file |
| `disconnect` | End a device's session |
| `adbcollect` | Run a forensic acquisition (see [Forensics](#forensics-censorship-circumvention-and-security-research)) |
| `adbdisable` | Turn off developer mode on the device |
| `adbclean` | Clear shell-accessible logs left by a session |

Run `help` at the prompt for flags.

## How it works

MESHadb is built on [MESHnet](https://github.com/BARGHEST-ngo/MESHnet), our small Go library for connecting a server to its clients over an encrypted, peer-to-peer tunnel.

```
  your computer                                           the phone
┌────────────────┐   1. direct: WireGuard over UDP     ┌────────────────┐
│ adb ─► meshcli │ ◄═════════════════════════════════► │  MESHadb app   │
│        bridge  │                                     │  ─► adbd       │
└───────┬────────┘                                     └────────┬───────┘
        │         2. fallback: HTTPS relay (TCP 443)            │
        └──────────────────►  relay  ◄──────────────────────────┘
                     (sees encrypted packets only)
```

### Peer-to-peer by default

MESHadb has no central server that carries your traffic. Both ends first reach each other through a relay, use it to exchange their network endpoints, and then punch through NAT to a direct UDP path. Once that path is up, ADB traffic flows straight between your computer and the phone.

### HTTPS relay as the fallback

Direct UDP is not always possible. Carrier-grade NAT, hotel and corporate Wi-Fi, and censored networks often block or degrade it. A plain WireGuard VPN simply stops working there. MESHadb keeps the session alive by sending the same encrypted packets through a [DERP](https://github.com/tailscale/tailscale/tree/main/cmd/derper#derp) relay over HTTPS on TCP port 443.

- The relay is a fallback and a meeting point. It is not a hub, and the connection moves to a direct path whenever one becomes available.
- Traffic is WireGuard-encrypted end to end between the CLI and the phone. The relay cannot decrypt it.
- To the network in between, a relayed session is an HTTPS connection.

### A session, step by step

1. `meshcli` starts a MESHnet server and prints its address. Each start gets a new key and a new address.
2. `pin` creates a 6-digit PIN that admits exactly one device.
3. The app joins with the address and the PIN. A device with only the address cannot connect.
4. The two ends connect directly, or through the relay if they cannot.
5. `adbpair` opens a local port and bridges it, through the tunnel, to the phone's wireless debugging port. `adb` connects to that local port.

## Forensics, Censorship Circumvention and Security Research

MESHadb is developed primarily for remote logical acquisition of forensic artifacts and for network monitoring, in support of forensics, censorship circumvention and security research. We built it to make spyware forensics and security research possible on devices in other countries.

Examples of research it supports:

- **Spyware forensics**: consensual acquisition from the device of a person at risk, wherever they are
- **Censorship monitoring**: observe blocking and interference from inside the affected network
- **Censorship circumvention**: give a device on a restricted network a way out through a machine you trust
- **Carrier services research**: study operator behaviour, preinstalled apps and network configuration on real devices in-country
- **App and traffic analysis**: capture what an app sends from a device on its real network

**Acquisition**

```
meshcli> adbcollect
meshcli> adbcollect --module BackupTar
meshcli> adbcollect --output /home/mesh/acquisitions/case-01
```

`adbcollect` runs an ADB acquisition using [AndroidQF](https://github.com/mvt-project/androidqf) and WARD. The output is in the format WARD and MVT expect. Acquisitions are written to `~/mesh-acquisitions` on your computer; set `ACQUISITION_DIR` in `.env` to change that.

**Exit node**

```
meshcli> forward on
```

`forward on` makes your machine the device's exit node: its TCP and UDP traffic travels through the encrypted tunnel and reaches the internet from your network. `forward off` returns it to its own connection. This gives you two things:

- **Capture**: the device's traffic passes through your machine, where you can record and inspect it. `tcpdump` and Suricata ship in the image; open `task shell` to run them.  
Alternatively run `capture start` to save pcap data to the device's acquisition directory
- **Circumvention**: a device on a censored or hostile network reaches the internet from your location. The local network sees only the tunnel, which falls back to HTTPS when UDP is blocked.

**After the session**

- `adbdisable` turns developer mode off, so the device is not left open to non-consensual extraction.
- `adbclean` clears logcat buffers and shell-accessible telemetry. Run it only after the acquisition, or you will destroy evidence. It is best effort: the paired ADB key and root-only logs remain on the device.

## Security model

- **The address is a secret.** Share the address and the PIN over separate channels, at least one of them end-to-end encrypted, such as Signal.
- **PINs are single-use.** One PIN admits one device. Three wrong guesses cancel the pairing.
- **Nothing persists.** A lost connection is a finished session. The device must rejoin with a new PIN.
- **Consent.** The person holding the phone installs the app, enters the PIN and enables Wireless debugging themselves. Only use MESHadb on devices whose owner has agreed.

## Repository structure

- `analyst`: the `meshcli` command-line client
- `android-client`: the Android app
- `docs`: documentation source

To build the Android app from source, run `make -C android-client apk`.

## Supported by

<img width="312" height="109" alt="Open Technology Fund" src="https://github.com/user-attachments/assets/d9476c43-7414-46e0-b6a5-1956bd6ca1b5" />

MESHadb is supported by the [Open Technology Fund](https://www.opentech.fund/) to support the consensual remote forensics of spyware and surveillance targeting Human Rights Defenders.

## License

MESHadb is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE) (AGPL-3.0-or-later). All modifications and additions by BARGHEST are Copyright (c) BARGHEST.

Portions are derived from [AndroidQF](https://github.com/mvt-project/androidqf) under the [MVT License 1.1](https://license.mvt.re/1.1/). Third-party components keep their own copyrights and licenses.
