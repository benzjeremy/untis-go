# 🚀 untis-go

Ein schneller, nativer und moderner WebUntis-Client für Schüler und Lehrer – geschrieben in **Go**. 

Vergiss langsame Web-Interfaces, unübersichtliche Oberflächen oder ressourcenfressende Apps. `untis-go` bringt deinen Stundenplan, Vertretungspläne und Schulinfos blitzschnell und plattformübergreifend auf deinen Desktop.

---

## ✨ Features

- **Blitzschnell & leichtgewichtig:** Dank Go als Single-Binary hast du minimale Startzeiten und einen extrem geringen RAM-Verbrauch (kein Electron-Bloat!).
- **Cross-Platform:** Läuft nahtlos auf **Windows und Linux (Weitere Systeme wie MacOS folgen)**.
- **Offizielle API:** Nutzt direkt die offiziellen Endpunkte der Untis-App für eine stabile und zuverlässige Datenanbindung.
- **Für alle optimiert:** Maßgeschneiderte Ansichten und Funktionen für Schüler und Lehrer.
- **Open Source:** Entwickelt mit KI-Assistenz und Community-Fokus unter der strengen GPLv3-Lizenz.

---

## 🛠️ Tech-Stack

- **Sprache:** [Go (Golang)](https://golang.org/)
- **Architektur:** Native Desktop-Applikation
- **API:** WebUntis API (offizielle Endpunkte)

---

## 📦 Installation & Nutzung

*(Anweisungen folgen, sobald die ersten Releases verfügbar sind)*

### Aus dem Quellcode bauen
Stelle sicher, dass du Go (ab Version 1.21+) auf deinem System installiert hast.

```bash
# Repository klonen
git clone [https://github.com/benzjeremy/untis-go.git](https://github.com/benzjeremy/untis-go.git)
cd untis-go

# Abhängigkeiten laden und bauen
go build -o untis-go main.go

# Starten
./untis-go
