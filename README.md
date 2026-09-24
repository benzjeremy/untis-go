# 🗓️ untis-go

[![Go Reference](https://pkg.go.dev/badge/github.com/benzjeremy/untis-go.svg)](https://pkg.go.dev/github.com/benzjeremy/untis-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/benzjeremy/untis-go.svg)](https://goreportcard.com/report/github.com/benzjeremy/untis-go)
[![CI](https://github.com/benzjeremy/untis-go/actions/workflows/ci.yml/badge.svg)](https://github.com/benzjeremy/untis-go/actions)
[![Release](https://img.shields.io/github/v/release/benzjeremy/untis-go)](https://github.com/benzjeremy/untis-go/releases/latest)
[![Status: Pre-Release](https://img.shields.io/badge/Status-Pre--Release%20%2F%20WIP-orange.svg)](https://github.com/benzjeremy/untis-go)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows-lightgrey)](#installation)

> 🌐 **Official Website:** [https://pi5.darter-basking.ts.net/untis-go/](https://pi5.darter-basking.ts.net/untis-go/)
> 📖 **Official Wiki & Documentation:** [https://pi5.darter-basking.ts.net/untis-go/wiki/](https://pi5.darter-basking.ts.net/untis-go/wiki/)

> [!IMPORTANT]
> ### 🚧 Pre-Release / Active Development Notice
> **This software is not yet finished and is actively being worked on.**  
> All versions (including `v2.3`) are **Pre-Releases** (Work in Progress), even if not originally announced as such. Active development, architectural refinements, and feature updates are ongoing.  
> If you encounter **errors, display bugs, or unexpected behavior**, please open an issue directly under [**GitHub Issues**](https://github.com/benzjeremy/untis-go/issues)! Feedback and pull requests are warmly welcome.

---

## 🎯 What is Untis GO?

A fast, native, and modern **WebUntis PC desktop client for students and teachers** – written in Go with a sleek desktop shell (inspired by clean sidebars of apps like Discord and Spotify Desktop).

Forget slow web interfaces, cluttered layouts, or resource-heavy web wrappers. **untis-go** brings your timetable, substitution schedules, homework, announcements, and absences lightning-fast and cross-platform to your desktop – **without Electron bloat!**

---

## ✨ Features

- 🏠 **Overview (Dashboard)**:
  - Personal greeting with name and school.
  - Direct display of next upcoming lesson and daily timeline.
  - Quick overview of open homework assignments and unread messages.
- 📅 **My Timetable**:
  - Personal student or teacher timetable.
  - Day and week view with **red LIVE-TIMELINE**.
  - Click on any lesson opens detailed information (teachers, substitutions, rooms, subject, homework).
- 👥 **Other Timetables**:
  - Quick switching between **all classes** (with live search over hundreds of classes), **teachers**, and **classrooms**.
- 📝 **Homework Management**:
  - Overview of all homework from WebUntis plus manually added tasks.
  - Tick off tasks, filter, and add new homework via a modal dialog.
- 🩺 **Absences**:
  - Absence list with status (Excused / Unexcused), period, and reason.
  - Enter new absences / sick notes.
- 💬 **Messages (Message Center)**:
  - Full-featured inbox for all official school messages, parent letters, and teacher announcements with full-text display.
- ⚙️ **Schools & Profile Management**:
  - Manage any number of schools and user profiles in parallel.
  - **Delete schools**: Each profile can be removed with a single click.
  - **Live school search**: Find your school worldwide via the official WebUntis school search.
- 🛡️ **100% Local-First & Privacy Compliant (GDPR / DSGVO)**:
  - All data stays strictly local on your machine in an encrypted SQLite database (`~/.local/share/untis-go/untis.db`).
  - Zero telemetry, zero tracking, zero external third-party cloud dependencies.
  - Full compliance with European GDPR (DSGVO) and State Data Protection Authority (LDI NRW) standards.
- 📅 **RFC-5545 iCalendar (.ics) Export**:
  - 1-click timetable export for Google Calendar, Apple Calendar & Mozilla Thunderbird.
  - Generates standard-compliant VEVENT blocks with room, teachers, notes, and cancellation status.
- 🔔 **Background Sync & Desktop Notifications**:
  - Automatic background monitoring daemon detecting timetable mutations (room changes, cancellations, substitutions).
  - Native OS desktop notifications (via Linux `notify-send` and Windows toast).
- 🖥️ **Native System Tray Integration**:
  - Minimize window to system tray / taskbar instead of quitting.
  - Quick menu for one-click schedule access, force refresh, status check, and graceful exit.

---

## 🔒 Security & Privacy (Zero-Telemetry)

- **100% Local-First & Privacy (GDPR / LDI NRW)**: Zero telemetry, zero tracking. All communication occurs directly with your school's WebUntis instance.
- **AES-256-GCM Encryption**: Credentials and cached secrets are never stored in plain text.
- **SQLite Cache**: Timetables load in under 1 ms directly from local storage.
- **Random Port & Crypto Session Token**: Protection against unauthorized local access (Strict Anti-DNS-Rebinding & Anti-CSRF).

---

## 📦 Installation & Download

### 1. Download Precompiled Packages (Recommended)

Download the matching file for your operating system from the [**Releases**](https://github.com/benzjeremy/untis-go/releases) page:

- **Linux (x86_64)**:
  ```bash
  tar -xzf untis-go-v2.3-linux.tar.gz
  sudo cp untis-go /usr/local/bin/
  untis-go
  ```
- **Windows (x86_64)**:
  - Unzip `untis-go-v2.3-windows.zip` and run `untis-go.exe`.

### 2. Installation via Go (`go install`)

If you have Go (version 1.21 or newer) installed:

```bash
go install github.com/benzjeremy/untis-go@latest
```

The binary will be compiled automatically into your `$GOPATH/bin` (or `~/go/bin`) and can be called directly as `untis-go` in your terminal.

### 3. Compile from Source

#### Prerequisites (Linux):
- Go 1.21+
- GTK 3 & WebKitGTK development libraries:
  - **Arch Linux / CachyOS**: `sudo pacman -S webkit2gtk-4.1 gtk3 gcc`
  - **Ubuntu / Debian**: `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential`
  - **Fedora**: `sudo dnf install webkit2gtk4.1-devel gtk3-devel gcc`

#### Building:
```bash
# 1. Clone the repository
git clone https://github.com/benzjeremy/untis-go.git
cd untis-go

# 2. Build binary
go build -o untis-go .

# 3. Launch
./untis-go
```

---

## ⌨️ Keyboard Shortcuts

| Key | Action |
|---|---|
| `←` / `→` | Previous / Next day (or week) |
| `T` | Jump to **Today** |
| `D` | Activate **Day view** |
| `W` | Activate **Week view** |
| `Esc` | Close open dialogs, info-sheets, and menus |

---

## 🐛 Bug Reports & Contributing

Bug reports and contributions are welcome:
1. Open the [**Issues**](https://github.com/benzjeremy/untis-go/issues) tab.
2. Click **New Issue**.
3. Briefly describe:
   - Operating system and desktop environment.
   - Steps to reproduce.
   - Expected vs actual behavior (including console log output).

---

## 📚 Wiki & Documentation

Detailed documentation and step-by-step guides are available in our official web wiki:  
👉 **[untis-go Wiki: https://pi5.darter-basking.ts.net/untis-go/wiki/](https://pi5.darter-basking.ts.net/untis-go/wiki/)**

- **Getting Started & Onboarding**: [Getting Started Guide](https://pi5.darter-basking.ts.net/untis-go/wiki/#quickstart)
- **Feature Deep Dive**: [Timetables, Homework & Absences](https://pi5.darter-basking.ts.net/untis-go/wiki/#features)
- **Installation**: [Linux & Windows Setup](https://pi5.darter-basking.ts.net/untis-go/wiki/#installation)
- **Security Architecture**: [AES-256-GCM & PBKDF2](https://pi5.darter-basking.ts.net/untis-go/wiki/#security)
- **FAQ & Troubleshooting**: [Frequently Asked Questions](https://pi5.darter-basking.ts.net/untis-go/wiki/#faq)

---

## 📄 License & Author

- **Developer:** Jeremy Benz ([@benzjeremy](https://github.com/benzjeremy))
- **License:** [GNU General Public License v3.0 (GPL-3.0)](LICENSE)
