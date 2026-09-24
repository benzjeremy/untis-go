# 📋 CHANGELOG – untis-go

Alle bedeutenden Änderungen an untis-go werden hier dokumentiert.  
Format: [Keep a Changelog](https://keepachangelog.com/de/1.0.0/) | Versionierung: SemVer ohne Trailing Zeros

---

## [v2.4] – 2026-09-25

### 🧹 Radikales Frontend-Cleanup (Zero Microsoft / Zero Telemetrie)
- **[CLEANUP] Vollständige Entfernung des Microsoft Gatekeepers & OneDrive-Modals:** Alle verbliebenen Dialoge, Device-Login-UI-Elemente und Styles wurden restlos aus `index.html` und `style.css` gelöscht.
- **[CLEANUP] Bereinigung des JavaScript-Clients:** Löschung aller Microsoft- und OneDrive-Funktionen, State-Felder und Window-Exports in `app.js` (~300 Zeilen toter Code eliminiert).
- **[PERF] Beschleunigter Start:** Der Web-Client startet direkt ohne überflüssige Gatekeeper-Prüfungen.
- **[SECURITY] 100% Local-First:** Absolut null Drittanbieter-Reste mehr im gesamten Quellcode.

## [v2.3] – 2026-09-10

### ✨ Neue Features
- **[FEAT] iCalendar Export (.ics):** Stundenplan direkt als Standard-.ics exportieren.
- **[FEAT] Hintergrund-Synchronisation & System-Tray:** Automatische Hintergrund-Aktualisierungen und native Tray-Icon-Unterstützung.

---

## [v2.2] – 2026-09-10

### 🔐 Sicherheit & Datenschutz
- **[PRIVACY] Microsoft-Login & OneDrive-Sync vollständig entfernt**
  - Grund: Datenschutzverstoß gegen LDI NRW / DSGVO Art. 5(1)(c), Art. 13, Art. 25 (Privacy by Design)
  - Betroffen war: `cloud/microsoft.go`, `cloud/sync.go`, 8 API-Endpunkte in `server/server.go`
  - Alternativer lokaler Backup-Mechanismus (AES-256-GCM, kein Drittanbieter) wird in Folgeversion implementiert

### ✨ Neue Features
- **[FEAT] Native Tray Icon Integration (Linux & Windows)** – Geplant

---

> ## ⚠️ Offizielle Entschuldigung & Transparenzerklärung
>
> **Betreffend:** Microsoft-OAuth2 / Microsoft-Login / OneDrive-Sync in untis-go v1.6
>
> Hiermit erkläre ich, Jeremy Benz, als Entwickler von untis-go, transparent und ehrlich:
>
> In **Version v1.6** wurde eine Integration des **Microsoft Entra ID (Azure AD) OAuth2-Logins** sowie eines **OneDrive-Konfigurationssyncs** in untis-go eingebaut. Dies war ein Fehler, der nicht hätte passieren dürfen.
>
> ### Was war das Problem?
> Die Integration sendete bei Anmeldung folgende Daten an Microsoft-Server:
> - Microsoft-Account-E-Mail-Adresse
> - Microsoft-Profilname
> - OAuth2-Access- und Refresh-Tokens
> - Konfigurationsdaten (via OneDrive-API)
>
> Dies widerspricht grundlegend dem Kern-Prinzip von untis-go:
> **100% Local-First, Zero Telemetrie, keine externen Drittanbieter-Dienste.**
>
> ### Warum ist das ein Problem?
> - **DSGVO Art. 5(1)(c) – Datensparsamkeit:** Für den WebUntis-Betrieb sind Microsoft-Daten nicht notwendig.
> - **DSGVO Art. 13 – Informationspflicht:** Nutzer wurden nicht ausreichend über die Datenweitergabe an Microsoft informiert.
> - **DSGVO Art. 25 – Privacy by Design:** Externe OAuth2-Dienste ohne Notwendigkeit verstoßen gegen dieses Prinzip.
> - **LDI NRW:** Als Entwickler aus Nordrhein-Westfalen unterliege ich den Richtlinien der Landesbeauftragten für Datenschutz und Informationsfreiheit NRW.
>
> ### Was wurde getan?
> - Der gesamte Microsoft-OAuth2-Code (`cloud/microsoft.go`, `cloud/sync.go`) wurde **vollständig entfernt**.
> - Alle 8 Microsoft-Auth-API-Endpunkte wurden **restlos aus dem Server entfernt**.
> - Eine **datenschutzkonforme lokale Alternative** (verschlüsseltes Backup ohne Drittanbieter) ist in Planung.
>
> ### Entschuldigung
> Ich entschuldige mich aufrichtig bei allen Nutzern von untis-go, die Version v1.6 verwendet haben. Diese Funktionalität hätte niemals in der Anwendung landen dürfen. Transparenz und echter Datenschutz sind keine optionalen Features — sie sind der Kern meiner Arbeit.
>
> — Jeremy Benz (@benzjeremy), 2026-09-10

---

## [v2.1] – 2026-xx-xx
*Changelog wird bei Release-Erstellung befüllt (nach Release-Vorlage)*

---

## [v2.0] – 2026-xx-xx
*Changelog wird bei Release-Erstellung befüllt*

---

## [v1.6] – 2026-xx-xx
### ⚠️ Enthielt Microsoft-Login (mittlerweile entfernt — siehe Entschuldigung oben)
- Hinzugefügt: Microsoft Entra ID OAuth2 Login
- Hinzugefügt: OneDrive-Konfigurationssync
- *(Diese Funktionalität wurde aus Datenschutzgründen in der Folgeversion vollständig entfernt)*

---

## [v1.5.2] – Hotfix
- [FIX] Wayland-Kompatibilitäts-Fix

## [v1.5.1] – Hotfix
- [FIX] Diverse Stabilitäts-Fixes

## [v1.5] – Feature Release
- [FEAT] Diverses

---

*Ältere Versionen: Siehe [GitHub Releases](https://github.com/benzjeremy/untis-go/releases)*
