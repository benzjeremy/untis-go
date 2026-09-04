# Erste Schritte mit Untis GO

Willkommen zu Untis GO! Dieser Leitfaden hilft dir dabei, die Anwendung einzurichten und zu verwenden.

## Systemanforderungen

- **Linux**: GTK 3 & WebKitGTK Entwicklungsdateien
  - Arch Linux / CachyOS: `sudo pacman -S webkit2gtk-4.1 gtk3 gcc`
  - Ubuntu / Debian: `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential`
  - Fedora: `sudo dnf install webkit2gtk4.1-devel gtk3-devel gcc`

- **Windows**: Keine zusätzlichen Abhängigkeiten erforderlich

## Installation

### Option 1: Vorkompilierte Pakete (Empfohlen)

1. Lade das passende Paket für dein Betriebssystem von der [Releases-Seite](https://github.com/benzjeremy/untis-go/releases) herunter:
   - Linux: `untis-go-linux-amd64.tar.gz`
   - Windows: `untis-go-windows-amd64.zip`

2. Entpacke das Archiv
3. Kopiere die ausführbare Datei in dein lokales Bin-Verzeichnis:
   - Linux: `sudo cp untis-go /usr/local/bin/`
   - Windows: Kopiere `untis-go.exe` nach `C:\Windows\System32\` oder füge den Ordner deinem PATH hinzu

4. Starte die Anwendung:
   - Linux: `untis-go`
   - Windows: Doppelklick auf `untis-go.exe`

### Option 2: Installation über Go

Wenn du Go (Version 1.21 oder neuer) installiert hast:

```bash
go install github.com/benzjeremy/untis-go@latest
```

Das Programm wird automatisch in dein `$GOPATH/bin` kompiliert und kann direkt als `untis-go` im Terminal aufgerufen werden.

### Option 3: Aus dem Quellcode kompilieren

1. Repository klonen:
   ```bash
   git clone https://github.com/benzjeremy/untis-go.git
   cd untis-go
   ```

2. Abhängigkeiten laden & bauen:
   ```bash
   go build -o untis-go .
   ```

3. Starten:
   ```bash
   ./untis-go
   ```

## Erststart & Onboarding

Beim ersten Start wirst du durch den Onboarding-Prozess geführt:

1. **Schule finden**: Nutze die integrierte WebUntis-Schulsuche, um deine Schule zu finden
2. **Anmelden**: Gib deine Benutzername und Passwort ein
3. **Profil speichern**: Deine Zugangsdaten werden sicher mit AES-256-GCM verschlüsselt lokal gespeichert
4. **Starten**: Nach erfolgreichem Login gelangst du zum Dashboard

## Wichtige Tastenkombinationen

| Taste | Aktion |
|-------|--------|
| ← / → | Vorheriger / Nächster Tag (bzw. Woche) |
| T | Zu Heute springen |
| D | Tagesansicht aktivieren |
| W | Wochenansicht aktivieren |
| Esc | Offene Dialoge, Info-Sheets und Menüs schließen |

## Unterstützung finden

- Dokumentation: Dieses Wiki
- Issue Tracker: [GitHub Issues](https://github.com/benzjeremy/untis-go/issues)
- Releases: [GitHub Releases](https://github.com/benzjeremy/untis-go/releases)

Viel Spaß mit Untis GO!