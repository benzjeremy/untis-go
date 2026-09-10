# 🚀 untis-go

[![Go Reference](https://pkg.go.dev/badge/github.com/benzjeremy/untis-go.svg)](https://pkg.go.dev/github.com/benzjeremy/untis-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/benzjeremy/untis-go.svg)](https://goreportcard.com/report/github.com/benzjeremy/untis-go)
[![CI](https://github.com/benzjeremy/untis-go/actions/workflows/ci.yml/badge.svg)](https://github.com/benzjeremy/untis-go/actions)
[![Coverage](https://codecov.io/gh/benzjeremy/untis-go/branch/main/graph/badge.svg)](https://app.codecov.io/gh/benzjeremy/untis-go)
[![Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#other-software)
[![Release: v2.3](https://img.shields.io/badge/Release-v2.3-orange.svg?style=for-the-badge&logo=github)](https://github.com/benzjeremy/untis-go/releases)
[![Status: Release](https://img.shields.io/badge/Status-RELEASE-green.svg?style=for-the-badge)](https://github.com/benzjeremy/untis-go/issues)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg?style=for-the-badge)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=for-the-badge&logo=go)](https://golang.org)
[![Security: AES-256-GCM](https://img.shields.io/badge/Security-AES--256--GCM-success.svg?style=for-the-badge&logo=lock)](https://en.wikipedia.org/wiki/Galois/Counter_Mode)

> ⚠️ **IMPORTANT NOTE: ENGLISH + GERMAN**  
> This README is provided in both English and German.  
> Select your preferred language above the content.

---

## 🇬🇧 English

> ⚠️ **IMPORTANT NOTICE: FIRST RELEASE VERSION**  
> **This is an early RELEASE version.** The application is actively developed and optimized.  
> If you encounter **errors, display bugs, or unexpected behavior**, please create a ticket directly under [**GitHub Issues**](https://github.com/benzjeremy/untis-go/issues)! Feedback, bug reports, and pull requests are expressly welcome and help improve the app.

### What is Untis GO?

A fast, native, and modern **WebUntis PC-Desktop client for students and teachers** – written in Go with a modern desktop shell (inspired by clean sidebars of apps like Discord and Spotify Desktop).

Forget slow web interfaces, cluttered layouts, or resource-heavy apps. **untis-go** brings your timetable, substitution schedules, homework, announcements, and absences lightning‑fast and cross‑platform to your desktop – **without Electron bloat!**

---

## 🇩🇪 Deutsch

> ⚠️ **WICHTIGER HINWEIS: ERSTE RELEASE‑VERSION**  
> **Dies ist eine frühe RELEASE‑Version.** Die Anwendung wird aktiv weiterentwickelt und optimiert.  
> Sollten dir **Fehler, Darstellungs‑Bugs oder unerwartetes Verhalten** auffallen, erstelle bitte direkt ein Ticket unter [**GitHub Issues**](https://github.com/benzjeremy/untis-go/issues)! Feedback, Fehlerberichte und Pull Requests sind ausdrücklich erwünscht und helfen, die App zu verbessern.

### Was ist Untis GO?

Ein schneller, nativer und moderner **WebUntis PC‑Desktop‑Client für Schüler und Lehrkräfte** – geschrieben in Go mit moderner Desktop‑Shell (angelehnt an cleane Sidebars moderner Desktop-Apps wie Discord und Spotify Desktop).

Vergiss langsame Web‑Interfaces, unübersichtliche Oberflächen oder ressourcenfressende Apps. **untis-go** bringt deinen Stundenplan, Vertretungspläne, Hausaufgaben, Mitteilungen und Abwesenheiten blitzschnell und plattformübergreifend auf deinen Desktop – **ohne Electron‑Bloat!**

---

## ✨ Features

### 🇬🇧 English

- 🏠 **Overview (Dashboard)**:
  - Personal greeting with name and school.
  - Direct display of the next lesson and today's schedule.
  - Quick overview of open homework and unread announcements/messages.
- 📅 **My Timetable**:
  - Your personal student or teacher timetable.
  - Day and week view with **red LIVE‑TIMELINE**.
  - Click on any lesson opens detailed information (teachers, substitutions, rooms, subject, homework).
- 👥 **Other Timetables**:
  - Quick switching between **all classes** (with live search over hundreds of classes), **teachers**, and **classrooms**.
- 📝 **Homework Management**:
  - Overview of all homework from WebUntis plus manually added tasks.
  - Tick off tasks, filter, and add new homework via a dialog.
- 🩺 **Absences**:
  - Absence list with status (Excused / Unexcused), period, and reason.
  - Enter new absences / sick notes.
- 💬 **Messages (Message Center)**:
  - Full‑featured inbox for all official school messages, parent letters, and teacher announcements with full‑text display.
- ⚙️ **Schools & Profile Management**:
  - Manage any number of schools and user profiles in parallel.
  - **Delete schools**: Each profile can be deleted with a click on the red trash‑can icon.
  - **Live school search**: Find your school worldwide via the official WebUntis school search – no pre‑configured default school!
- 🛡️ **100% Local-First & Privacy Compliant (GDPR / DSGVO)**:
  - All data stays strictly local on your machine in an encrypted SQLite database (`~/.local/share/untis-go/untis.db`).
  - Zero telemetry, zero tracking, zero external third-party cloud dependencies.
  - Full compliance with European GDPR (DSGVO) and State Data Protection Authority (LDI NRW) standards.
- 📅 **RFC-5545 iCalendar (.ics) Export**:
  - 1-click timetable export for Google Calendar, Apple Calendar & Mozilla Thunderbird.
  - Generates standard-compliant VEVENT blocks with room, teachers, notes, and cancellation status.
- 🔔 **Background Sync & Desktop Notifications**:
  - Automatic background monitoring daemon detecting timetable mutations (room changes, cancellations, substitutions).
  - Native OS desktop notifications (via Linux notify-send and Windows toast).
- 🖥️ **Native System Tray Integration**:
  - Minimize window to system tray / taskbar instead of quitting.
  - Quick menu for one-click schedule access, force refresh, status check, and graceful exit.

### 🇩🇪 Deutsch

- 🏠 **Übersicht (Dashboard)**:
  - Persönliche Begrüßung mit Name und Schule.
  - Direktanzeige der nächsten Unterrichtsstunde und des heutigen Tagesablaufs.
  - Schnelle Übersicht über offene Hausaufgaben und ungelesene Durchsagen/Mitteilungen.
- 📅 **Mein Stundenplan**:
  - Dein persönlicher Schüler‑ oder Lehrer‑Stundenplan.
  - Tages‑ und Wochenansicht mit **roter JETZT‑Live‑Zeitlinie**.
  - Klick auf jede Unterrichtsstunde öffnet Detailinfos (Lehrkräfte, Vertretungen, Räume, Lehrstoff, Hausaufgaben).
- 👥 **Weitere Stundenpläne**:
  - Schneller Wechsel zwischen **allen Klassen** (mit Live‑Suchfilter über hunderte Klassen), **Lehrkräften** und **Fachräumen**.
- 📝 **Hausaufgaben‑Verwaltung**:
  - Übersicht aller Hausaufgaben aus WebUntis sowie manuell hinzugefügter Aufgaben.
  - Aufgaben abhaken, filtern und neue Hausaufgaben über einen Dialog hinzufügen.
- 🩺 **Abwesenheiten**:
  - Fehlzeitenliste mit Status (Entschuldigt / Unentschuldigt), Zeitraum und Begründung.
  - Eintragen neuer Abwesenheiten / Krankmeldungen.
- 💬 **Mitteilungen (Nachrichtenzentrum)**:
  - Vollwertiger Posteingang für alle offiziellen Schulnachrichten, Elternbriefe und Lehrerdurchsagen mit Volltext‑Anzeige.
- ⚙️ **Schulen‑ & Profilverwaltung**:
  - Beliebig viele Schulen und Benutzerprofile parallel verwalten.
  - **Schulen löschen**: Jedes Profil kann mit einem Klick auf das rote Papierkorb‑Icon gelöscht werden.
  - **Live‑Schulsuche**: Finde deine Schule weltweit über die offizielle WebUntis‑Schulsuche – keine feste Standardschule vorkonfiguriert!
- 🛡️ **100% Local-First & Datenschutz-konform (DSGVO / LDI NRW)**:
  - Alle Daten bleiben strikt lokal auf deinem Rechner in einer verschlüsselten SQLite-Datenbank (`~/.local/share/untis-go/untis.db`).
  - Keine Telemetrie, kein Tracking, keinerlei externe Drittanbieter-Cloud-Abhängigkeiten.
  - Volle Einhaltung der Datenschutz-Grundverordnung (DSGVO) und Richtlinien der LDI NRW.
- 📅 **RFC-5545 iCalendar (.ics) Export**:
  - 1-Klick-Export des Stundenplans für Google Calendar, Apple Calendar & Mozilla Thunderbird.
  - Standardkonforme VEVENT-Blöcke inklusive Räume, Lehrkräfte, Notizen und Entfall-Status.
- 🔔 **Hintergrund-Synchronisation & Desktop-Benachrichtigungen**:
  - Automatischer Hintergrund-Daemon zur Erkennung von Stundenplan-Änderungen (Raumänderungen, Entfällen, Vertretungen).
  - Native Desktop-Benachrichtigungen des Betriebssystems (Linux notify-send & Windows Toast).
- 🖥️ **Native System-Tray Integration**:
  - Minimieren in die Taskleiste statt Schließen der Anwendung.
  - Tray-Menü für Schnellzugriff auf Stundenplan, manuelles Neuladen, Statusanzeige und Beenden.

---

## 🔒 Security & Privacy (Zero‑Telemetry)

### 🇬🇧 English

- **100% Local-First & Privacy (GDPR / LDI NRW)**: Zero telemetry, zero tracking. All communication occurs directly with your school's WebUntis instance.
- **AES‑256‑GCM Encryption**: Credentials and cached secrets are never stored in plain text.
- **SQLite Cache**: Timetables load in under 1 ms directly from local storage.
- **Random Port & Crypto Session Token**: Protection against unauthorized local access (Strict Anti-DNS-Rebinding & Anti-CSRF).

### 🇩🇪 Deutsch

- **100% Local-First & Datenschutz (DSGVO / LDI NRW)**: Keine Telemetrie, kein Tracking. Direkte Kommunikation ausschließlich mit dem WebUntis-Server deiner Schule.
- **AES‑256‑GCM Verschlüsselung**: Zugangsdaten und Cache-Secrets werden niemals im Klartext gespeichert.
- **SQLite Cache**: Stundenpläne laden in unter 1 Millisekunde direkt aus dem lokalen Speicher.
- **Zufallsport & Krypto-Session-Token**: Schutz vor unbefugtem lokalen Zugriff (Strikter Schutz gegen DNS-Rebinding & CSRF).

---

## 📦 Installation & Download

### 🇬🇧 English

#### 1. Download pre‑compiled packages (Recommended)

Download the matching file for your operating system from the [**Releases**](https://github.com/benzjeremy/untis-go/releases) page:

- **Linux (x86_64)**:
  ```bash
  tar -xzf untis-go-v2.2-linux.tar.gz
  sudo cp untis-go /usr/local/bin/
  untis-go
  ```
- **Windows (x86_64)**:
  - Unzip `untis-go-v2.2-windows.zip` and start `untis-go.exe`.

#### 2. Installation via Go (`go install`)

If you have Go (version 1.21 or newer) installed on your computer, you can install `untis-go` system‑wide with a single command:

```bash
go install github.com/benzjeremy/untis-go@latest
```

The program will be compiled automatically into your `$GOPATH/bin` (or `~/go/bin`) and can be called directly as `untis-go` in the terminal (provided `~/go/bin` is in your `$PATH`).

#### 3. Compile from source

##### Prerequisites (Linux):
- Go 1.21+
- GTK 3 & WebKitGTK development files:
  - **Arch Linux / CachyOS**: `sudo pacman -S webkit2gtk-4.1 gtk3 gcc`
  - **Ubuntu / Debian**: `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential`
  - **Fedora**: `sudo dnf install webkit2gtk4.1-devel gtk3-devel gcc`

##### Building:
```bash
# 1. Clone the repository
git clone https://github.com/benzjeremy/untis-go.git
cd untis-go

# 2. Load dependencies & build
go build -o untis-go .

# 3. Start
./untis-go
```

### 🇩🇪 Deutsch

#### 1. Vorkompilierte Pakete herunterladen (Empfohlen)

Lade die passende Datei für dein Betriebssystem unter [**Releases**](https://github.com/benzjeremy/untis-go/releases) herunter:

- **Linux (x86_64)**:
  ```bash
  tar -xzf untis-go-v2.2-linux.tar.gz
  sudo cp untis-go /usr/local/bin/
  untis-go
  ```
- **Windows (x86_64)**:
  - Entpacke `untis-go-v2.2-windows.zip` und starte `untis-go.exe`.

#### 2. Installation über Go (`go install`)

Wenn du Go (ab Version 1.21) auf deinem Rechner installiert hast, kannst du `untis-go` mit einem einzigen Befehl systemweit installieren:

```bash
go install github.com/benzjeremy/untis-go@latest
```

Das Programm wird automatisch in dein `$GOPATH/bin` (bzw. `~/go/bin`) kompiliert und kann direkt als `untis-go` im Terminal aufgerufen werden (sofern `~/go/bin` in deinem `$PATH` liegt).

#### 3. Aus dem Quellcode kompilieren

##### Voraussetzungen (Linux):
- Go 1.21+
- GTK 3 & WebKitGTK Entwicklungsdateien:
  - **Arch Linux / CachyOS**: `sudo pacman -S webkit2gtk-4.1 gtk3 gcc`
  - **Ubuntu / Debian**: `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential`
  - **Fedora**: `sudo dnf install webkit2gtk4.1-devel gtk3-devel gcc`

##### Bauen:
```bash
# 1. Repository klonen
git clone https://github.com/benzjeremy/untis-go.git
cd untis-go

# 2. Abhängigkeiten laden & bauen
go build -o untis-go .

# 3. Starten
./untis-go
```

---

## ⌨️ Keyboard Shortcuts (Tastenkombinationen)

| Key / Taste | Action / Aktion |
|-------------|-----------------|
| `←` / `→`   | Previous / Next day (or week) – Vorheriger / Nächster Tag (bzw. Woche) |
| `T`         | Jump to **Today** – Zu **Heute** springen |
| `D`         | Activate **Day view** – Tagesansicht aktivieren |
| `W`         | Activate **Week view** – Wochenansicht aktivieren |
| `Esc`       | Close open dialogs, info‑sheets, and menus – Offene Dialoge, Info‑Sheets und Menüs schließen |

---

## 🐛 Found a bug? (Create an issue)

### 🇬🇧 English

Since `untis-go` is currently in the **first RELEASE phase**, bug reports are extremely valuable!  
If something does not work as expected:

1. Open the [**Issues**](https://github.com/benzjeremy/untis-go/issues) tab.
2. Click **New Issue**.
3. Briefly describe:
   - Which operating system are you using?
   - What did you do?
   - What happened (error message, log output)?
4. Submit the ticket – we will look into it as soon as possible!

### 🇩🇪 Deutsch

Da sich `untis-go` aktuell in der **ersten RELEASE‑Phase** befindet, sind Fehlerberichte extrem wertvoll!  
Wenn etwas nicht wie erwartet funktioniert:

1. Öffne den Tab [**Issues**](https://github.com/benzjeremy/untis-go/issues).
2. Klicke auf **New Issue**.
3. Beschreibe kurz:
   - Welches Betriebssystem nutzt du?
   - Was hast du gemacht?
   - What happened (error message, log output)?
4. Ticket abschicken – wir schauen uns das so schnell wie möglich an!

---

## 📄 License

This project is licensed under the **GNU General Public License v3.0 (GPL‑3.0)**.  
See the [LICENSE](LICENSE) file for details.

---

## 🌐 Language / Sprachwahl

The application, website, and documentation are available in **English and German**.  
Select your preferred language in the app settings or via the language selector on the website.

**More languages coming soon** – Weitere Sprachen folgen

## 📚 Wiki & Dokumentation
 
Ausführliche Dokumentation, Schritt-für-Schritt-Anleitungen und Bildmaterial:
 
- 🌐 **Interaktives Web-Wiki**: [https://benzjeremy.github.io/untis-go/wiki/](https://benzjeremy.github.io/untis-go/wiki/)
- **GitHub Wiki Home**: https://github.com/benzjeremy/untis-go.wiki/wiki/Home
- **Getting Started**: https://github.com/benzjeremy/untis-go.wiki/wiki/Getting-Started
- **Features**: https://github.com/benzjeremy/untis-go.wiki/wiki/Features
- **Installation**: https://github.com/benzjeremy/untis-go.wiki/wiki/Installation
- **Security**: https://github.com/benzjeremy/untis-go.wiki/wiki/Security
- **FAQ**: https://github.com/benzjeremy/untis-go.wiki/wiki/FAQ

---