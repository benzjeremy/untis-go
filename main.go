package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/benzjeremy/untis-go/db"
	"github.com/benzjeremy/untis-go/server"
)

const AppVersion = server.AppVersion

func main() {
	portFlag := flag.Int("port", 0, "HTTP-Port für den lokalen Untis-Server (Standard: 0 für dynamischen Zufallsport)")
	noGuiFlag := flag.Bool("no-gui", false, "Startet nur den Webserver ohne GUI/Fenster")
	browserFlag := flag.Bool("browser", false, "Erzwingt das Öffnen im Webbrowser statt WebKitGTK")
	flag.Parse()

	log.Println("==================================================")
	log.Printf(" Untis Stundenplan-Anwendung (Go) v%s\n", AppVersion)
	log.Println(" Release: PC-Desktop Remake (GPL-3.0 Lizenz)")
	log.Println("==================================================")

	// 1. Initialize SQLite database & encryption at ~/.local/share/untis-go/untis.db
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("[Fehler] SQLite-Datenbank konnte nicht initialisiert werden: %v", err)
	}
	defer database.Close()

	// 2. Initialize local server with 32-character session token and database
	srv := server.NewServer(database)

	port := 0
	if *portFlag > 0 {
		port = *portFlag
	}

	serverURL, err := srv.Start(port)
	if err != nil {
		log.Fatalf("[Fehler] Desktop-Engine konnte nicht gestartet werden: %v", err)
	}

	log.Printf("[Desktop] Native Anwendungs-Engine v%s initialisiert (Port: %d, Loopback-isoliert)\n", AppVersion, srv.GetPort())
	log.Printf("[Sicherheit] Session-Token geschützt, DNS-Rebinding- & CSRF-Filter aktiv.")
	if *noGuiFlag || *browserFlag {
		log.Printf("[Web-Schnittstelle] URL: %s\n", serverURL)
	}

	activeProf, _ := database.GetActiveProfile()
	if activeProf != nil {
		log.Printf("[Untis] Aktives Profil: '%s' | Schule: %s | Server: %s\n", activeProf.Name, activeProf.School, activeProf.Server)
	} else {
		log.Println("[Untis] Kein aktives Profil – Onboarding 'Finde deine Schule' aktiv.")
	}

	// Channel for graceful termination
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	if *noGuiFlag {
		log.Println("[Info] Server-Modus ohne GUI aktiv. Beenden mit Strg+C.")
		<-shutdownChan
		log.Println("\n[Info] Fahre Server herunter...")
		_ = srv.Stop()
		log.Println("[Info] Beendet.")
		return
	}

	// Start GUI window in foreground or browser
	go func() {
		<-shutdownChan
		log.Println("\n[Info] Signal empfangen, beende Anwendung...")
		_ = srv.Stop()
		os.Exit(0)
	}()

	windowTitle := "Untis Stundenplan"
	if activeProf != nil && activeProf.School != "" {
		userName := activeProf.Name
		if activeProf.Username != "" {
			userName = activeProf.Username
		}
		windowTitle = fmt.Sprintf("Untis Stundenplan - %s - %s", activeProf.School, userName)
	}

	// Launch native WebKitGTK (hardware accelerated) or browser
	LaunchGUI(windowTitle, serverURL, 1150, 800, *browserFlag)

	// Clean up when GUI closes
	log.Println("[Info] Anwendungsfenster geschlossen. Fahre Server herunter...")
	_ = srv.Stop()
	log.Println("[Info] Auf Wiedersehen!")
}
