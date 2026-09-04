# 🚀 untis-go [BETA]

[![Go Reference](https://pkg.go.dev/badge/github.com/benzjeremy/untis-go.svg)](https://pkg.go.dev/github.com/benzjeremy/untis-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/benzjeremy/untis-go)](https://goreportcard.com/report/github.com/benzjeremy/untis-go)
[![CI](https://github.com/benzjeremy/untis-go/actions/workflows/ci.yml/badge.svg)](https://github.com/benzjeremy/untis-go/actions)
[![Coverage](https://codecov.io/gh/benzjeremy/untis-go/branch/main/graph/badge.svg)](https://app.codecov.io/gh/benzjeremy/untis-go)
[![Release: v1.0.0-beta.1](https://img.shields.io/badge/Release-v1.0.0--beta.1-orange.svg?style=for-the-badge&logo=github)](https://github.com/benzjeremy/untis-go/releases)
[![Status: Beta](https://img.shields.io/badge/Status-BETA-yellow.svg?style=for-the-badge)](https://github.com/benzjeremy/untis-go/issues)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg?style=for-the-badge)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=for-the-badge&logo=go)](https://golang.org)
[![Security: AES-256-GCM](https://img.shields.io/badge/Security-AES--256--GCM-success.svg?style=for-the-badge&logo=lock)](https://en.wikipedia.org/wiki/Galois/Counter_Mode)

> ⚠️ **WICHTIGER HINWEIS: BETA-VERSION**  
> **Dies ist eine frühe Beta-Version.** Die Anwendung wird aktiv weiterentwickelt und optimiert.  
> Sollten dir **Fehler, Darstellungs-Bugs oder unerwartetes Verhalten** auffallen, erstelle bitte direkt ein Ticket unter [**GitHub Issues**](https://github.com/benzjeremy/untis-go/issues)! Feedback, Fehlerberichte und Pull Requests sind ausdrücklich erwünscht und helfen, die App zu verbessern.

---

Ein schneller, nativer und moderner **WebUntis PC-Desktop-Client für Schüler und Lehrkräfte** – geschrieben in Go mit moderner Desktop-Shell (angelehnt an die Sidebar von Microsoft Teams, Discord und Spotify Desktop).

Vergiss langsame Web-Interfaces, unübersichtliche Oberflächen oder ressourcenfressende Apps. **untis-go** bringt deinen Stundenplan, Vertretungspläne, Hausaufgaben, Mitteilungen und Abwesenheiten blitzschnell und plattformübergreifend auf deinen Desktop – **ohne Electron-Bloat!**

---

## ✨ Features

- 🏠 **Übersicht (Dashboard)**:
  - Persönliche Begrüßung mit Name und Schule.
  - Direktanzeige der nächsten Unterrichtsstunde und des heutigen Tagesablaufs.
  - Schnelle Übersicht über offene Hausaufgaben und ungelesene Durchsagen/Mitteilungen.
- 📅 **Mein Stundenplan**:
  - Dein persönlicher Schüler- oder Lehrer-Stundenplan.
  - Tages- und Wochenansicht mit **roter JETZT-Live-Zeitlinie**.
  - Klick auf jede Unterrichtsstunde öffnet Detailinfos (Lehrkräfte, Vertretungen, Räume, Lehrstoff, Hausaufgaben).
- 👥 **Weitere Stundenpläne**:
  - Schneller Wechsel zwischen **allen Klassen** (mit Live-Suchfilter über hunderte Klassen), **Lehrkräften** und **Fachräumen**.
- 📝 **Hausaufgaben-Verwaltung**:
  - Übersicht aller Hausaufgaben aus WebUntis sowie manuell hinzugefügter Aufgaben.
  - Aufgaben abhaken, filtern und neue Hausaufgaben über einen Dialog hinzufügen.
- 🩺 **Abwesenheiten**:
  - Fehlzeitenliste mit Status (Entschuldigt / Unentschuldigt), Zeitraum und Begründung.
  - Eintragen neuer Abwesenheiten / Krankmeldungen.
- 💬 **Mitteilungen (Nachrichtenzentrum)**:
  - Vollwertiger Posteingang für alle offiziellen Schulnachrichten, Elternbriefe und Lehrerdurchsagen mit Volltext-Anzeige.
- ⚙️ **Schulen- & Profilverwaltung**:
  - Beliebig viele Schulen und Benutzerprofile parallel verwalten.
  - **Schulen löschen**: Jedes Profil kann mit einem Klick auf das rote Papierkorb-Icon gelöscht werden.
  - **Live-Schulsuche**: Finde deine Schule weltweit über die offizielle WebUntis-Schulsuche – keine feste Standardschule vorkonfiguriert!
- 🛡️ **Sicherheit & Zero-Lag**:
  - **AES-256-GCM Verschlüsselung**: Passwörter werden niemals im Klartext gespeichert.
  - **SQLite Cache**: Stundenpläne laden in unter 1 Millisekunde direkt aus dem lokalen Speicher.
  - **Zufallsport & Session-Token**: Schutz vor unbefugten lokalen Zugriffen.

---

## 📦 Installation & Download

### 1. Vorkompilierte Pakete herunterladen (Empfohlen)

Lade die passende Datei für dein Betriebssystem unter [**Releases**](https://github.com/benzjeremy/untis-go/releases) herunter:

- **Linux (x86_64)**:
  ```bash
  tar -xzf untis-go-linux-amd64.tar.gz
  sudo cp untis-go /usr/local/bin/
  untis-go
  ```
- **Windows (x86_64)**:
  - Entpacke `untis-go-windows-amd64.zip` und starte `untis-go.exe`.

---

### 2. Installation über Go (`go install`)

Wenn du Go (ab Version 1.21) auf deinem Rechner installiert hast, kannst du `untis-go` mit einem einzigen Befehl systemweit installieren:

```bash
go install github.com/benzjeremy/untis-go@latest
```

Das Programm wird automatisch in dein `$GOPATH/bin` (bzw. `~/go/bin`) kompiliert und kann direkt als `untis-go` im Terminal aufgerufen werden (sofern `~/go/bin` in deinem `$PATH` liegt).

---

### 3. Aus dem Quellcode kompilieren

#### Voraussetzungen (Linux):
- Go 1.21+
- GTK 3 & WebKitGTK Entwicklungsdateien:
  - **Arch Linux / CachyOS**: `sudo pacman -S webkit2gtk-4.1 gtk3 gcc`
  - **Ubuntu / Debian**: `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential`
  - **Fedora**: `sudo dnf install webkit2gtk4.1-devel gtk3-devel gcc`

#### Bauen:
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

## ⌨️ Tastenkombinationen (Shortcuts)

| Taste | Aktion |
|---|---|
| `←` / `→` | Vorheriger / Nächster Tag (bzw. Woche) |
| `T` | Zu **Heute** springen |
| `D` | **Tagesansicht** aktivieren |
| `W` | **Wochenansicht** aktivieren |
| `Esc` | Offene Dialoge, Info-Sheets und Menüs schließen |

---

## 🐛 Fehler gefunden? (Issue erstellen)

Da sich `untis-go` aktuell in der **BETA-Phase** befindet, sind Fehlerberichte extrem wertvoll!  
Wenn etwas nicht wie erwartet funktioniert:

1. Öffne den Tab [**Issues**](https://github.com/benzjeremy/untis-go/issues).
2. Klicke auf **New Issue**.
3. Beschreibe kurz:
   - Welches Betriebssystem nutzt du?
   - Was hast du gemacht?
   - Was ist passiert (Fehlermeldung, Log-Ausgabe)?
4. Ticket abschicken – wir schauen uns das so schnell wie möglich an!

---

## 📄 Lizenz

Dieses Projekt ist unter der **GNU General Public License v3.0 (GPL-3.0)** lizenziert.  
Weitere Informationen findest du in der Datei [LICENSE](LICENSE).
