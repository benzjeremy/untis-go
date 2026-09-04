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

const AppVersion = "1.0.0"

func main() {
	portFlag := flag.Int("port", 0, "HTTP-Port für den lokalen Untis-Server (Standard: 0 für dynamischen Zufallsport)")
	noGuiFlag := flag.Bool("no-gui", false, "Startet nur den Webserver ohne GUI/Fenster")
	browserFlag := flag.Bool("browser", false, "Erzwingt das Öffnen im Webbrowser statt WebKitGTK")
	flag.Parse()

	log.Println("==================================================")
	log.Printf(" Untis Stundenplan-Anwendung (Go) v%s\n", AppVersion)
	log.Println(" Release: PC-Desktop Remake (MIT-Lizenz)")
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
		log.Fatalf("[Fehler] Server konnte nicht gestartet werden: %v", err)
	}

	log.Printf("[Sicherheit] Dynamischer Port & 32-Zeichen-Session-Token aktiv.")
	log.Printf("[Server] Läuft unter: %s\n", serverURL)

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
		windowTitle = fmt.Sprintf("Untis Stundenplan - %s", activeProf.School)
		className := database.GetSetting("selected_class_name", "")
		if className != "" {
			windowTitle = fmt.Sprintf("Untis Stundenplan - %s (%s)", activeProf.School, className)
		}
	}

	// Launch native WebKitGTK (hardware accelerated) or browser
	LaunchGUI(windowTitle, serverURL, 1150, 800, *browserFlag)

	// Clean up when GUI closes
	log.Println("[Info] Anwendungsfenster geschlossen. Fahre Server herunter...")
	_ = srv.Stop()
	log.Println("[Info] Auf Wiedersehen!")
}
