package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benzjeremy/untis-go/api"
	"github.com/benzjeremy/untis-go/db"
	"github.com/benzjeremy/untis-go/updater"
	"github.com/benzjeremy/untis-go/web"
)

// AppVersion defines the current application version
const AppVersion = "1.3.1"

// Server coordinates the local HTTP API and SQLite database
type Server struct {
	database     *db.Database
	sessionToken string
	activeClient *api.Client
	mu           sync.RWMutex
	httpServer   *http.Server
	listener     net.Listener
	port         int
}

// NewServer initializes the server with SQLite database and a 32-character crypto session token
func NewServer(database *db.Database) *Server {
	token := generateCryptoToken(16) // 16 bytes = 32 hex chars

	s := &Server{
		database:     database,
		sessionToken: token,
	}

	// Initialize active client if an active profile exists
	if activeProf, err := database.GetActiveProfile(); err == nil && activeProf != nil {
		pwd, _ := database.GetDecryptedPassword(activeProf)
		client := api.NewClient(activeProf.Server, activeProf.School, activeProf.Username, pwd, "password")
		s.activeClient = client

		// Background authenticate so token is fresh
		go func() {
			if err := client.Authenticate(); err != nil {
				log.Printf("[WebUntis] Initialer Login-Hinweis für '%s': %v", activeProf.School, err)
			} else {
				log.Printf("[WebUntis] Angemeldet als %s (%s)", client.UserInfo.DisplayName, client.Username)
			}
		}()
	}

	return s
}

// generateCryptoToken generates a cryptographically secure hex string
func generateCryptoToken(bytesCount int) string {
	b := make([]byte, bytesCount)
	if _, err := rand.Read(b); err != nil {
		// Fallback
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GetSessionToken returns the active session token
func (s *Server) GetSessionToken() string {
	return s.sessionToken
}

// Start launches the HTTP server and returns the active URL with token
func (s *Server) Start(port int) (string, error) {
	mux := http.NewServeMux()

	// Wrap API routes with session token verification
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/status", s.handleStatus)
	apiMux.HandleFunc("/api/dashboard", s.handleDashboard)
	apiMux.HandleFunc("/api/profiles", s.handleProfiles)
	apiMux.HandleFunc("/api/profiles/switch", s.handleProfileSwitch)
	apiMux.HandleFunc("/api/profiles/delete", s.handleProfileDelete)
	apiMux.HandleFunc("/api/schools/search", s.handleSchoolSearch)
	apiMux.HandleFunc("/api/classes", s.handleClasses)
	apiMux.HandleFunc("/api/timetable", s.handleTimetable)
	apiMux.HandleFunc("/api/timetable/own", s.handleOwnTimetable)
	apiMux.HandleFunc("/api/timetable/resource", s.handleResourceTimetable)
	apiMux.HandleFunc("/api/teachers", s.handleTeachers)
	apiMux.HandleFunc("/api/rooms", s.handleRooms)
	apiMux.HandleFunc("/api/messages", s.handleMessages)
	apiMux.HandleFunc("/api/messages/", s.handleMessageDetail)
	apiMux.HandleFunc("/api/homework", s.handleHomework)
	apiMux.HandleFunc("/api/absences", s.handleAbsences)
	apiMux.HandleFunc("/api/settings", s.handleSettings)
	apiMux.HandleFunc("/api/settings/aliases", s.handleSubjectAliases)
	apiMux.HandleFunc("/api/refresh", s.handleRefresh)
	apiMux.HandleFunc("/api/updates/check", s.handleUpdateCheck)
	apiMux.HandleFunc("/api/updates/apply", s.handleUpdateApply)

	// Route dispatch with token verification for /api/
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Session-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" || token != s.sessionToken {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Unauthorized",
				"message": "Ungültiges oder fehlendes Session-Token",
			})
			return
		}

		apiMux.ServeHTTP(w, r)
	})

	// Static frontend routes (accessible locally)
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/", s.handleIndex)

	// Bind to dynamic random port on 127.0.0.1 (or specific port if provided)
	bindAddr := "127.0.0.1:0"
	if port > 0 {
		bindAddr = fmt.Sprintf("127.0.0.1:%d", port)
	}

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		// Fallback to random port
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("tcp-listener konnte nicht geöffnet werden: %w", err)
		}
	}

	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port
	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  25 * time.Second,
		WriteTimeout: 25 * time.Second,
	}

	go func() {
		if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[Server] Error: %v", err)
		}
	}()

	return fmt.Sprintf("http://127.0.0.1:%d/?token=%s", s.port, s.sessionToken), nil
}

// Stop terminates the server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// Handler: /api/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	activeProf, err := s.database.GetActiveProfile()
	if err != nil || activeProf == nil {
		// Needs Onboarding!
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"needsOnboarding": true,
			"theme":           s.database.GetSetting("theme", "dark"),
			"defaultView":     s.database.GetSetting("default_view", "day"),
		})
		return
	}

	selectedClassID := s.database.GetIntSetting("selected_class_id", 0)
	selectedClassName := s.database.GetSetting("selected_class_name", "")

	var displayName, email, detectedClass string
	var authenticated bool

	if client != nil {
		displayName = client.UserInfo.DisplayName
		email = client.UserInfo.Email
		detectedClass = client.UserInfo.DetectedClass
		authenticated = client.Token != ""
	}

	if displayName == "" {
		displayName = activeProf.Name
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needsOnboarding":   false,
		"activeProfileId":   activeProf.ID,
		"profileName":       activeProf.Name,
		"school":            activeProf.School,
		"server":            activeProf.Server,
		"username":          activeProf.Username,
		"displayName":       displayName,
		"email":             email,
		"detectedClass":     detectedClass,
		"selectedClassId":   selectedClassID,
		"selectedClassName": selectedClassName,
		"theme":             s.database.GetSetting("theme", "dark"),
		"defaultView":       s.database.GetSetting("default_view", "day"),
		"authenticated":     authenticated,
	})
}

// Handler: /api/profiles
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profiles, err := s.database.GetProfiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		activeProf, _ := s.database.GetActiveProfile()
		activeID := ""
		if activeProf != nil {
			activeID = activeProf.ID
		}

		type safeProfile struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			School    string `json:"school"`
			Server    string `json:"server"`
			Username  string `json:"username"`
			IsActive  bool   `json:"isActive"`
			CreatedAt string `json:"createdAt"`
		}

		var list []safeProfile
		for _, p := range profiles {
			list = append(list, safeProfile{
				ID:        p.ID,
				Name:      p.Name,
				School:    p.School,
				Server:    p.Server,
				Username:  p.Username,
				IsActive:  p.IsActive,
				CreatedAt: p.CreatedAt.Format(time.RFC3339),
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"profiles":        list,
			"activeProfileId": activeID,
		})

	case http.MethodPost:
		var req struct {
			ID        string `json:"id,omitempty"`
			Name      string `json:"name"`
			School    string `json:"school"`
			Server    string `json:"server"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			SetActive bool   `json:"setActive"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Ungültiger Anfragekörper"})
			return
		}

		req.School = strings.TrimSpace(req.School)
		req.Server = api.NormalizeServerURL(req.Server)
		req.Username = strings.TrimSpace(req.Username)

		if req.School == "" || req.Server == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Schule und Server-URL sind erforderlich"})
			return
		}

		// Test connection against WebUntis API
		testClient := api.NewClient(req.Server, req.School, req.Username, req.Password, "password")
		if err := testClient.Authenticate(); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Verbindung fehlgeschlagen: %v", err),
			})
			return
		}

		profID := req.ID
		if profID == "" {
			profID = fmt.Sprintf("%d", time.Now().Unix())
		}

		profName := req.Name
		if profName == "" {
			if testClient.UserInfo.DisplayName != "" {
				profName = fmt.Sprintf("%s (%s)", testClient.UserInfo.DisplayName, req.School)
			} else if req.Username != "" {
				profName = fmt.Sprintf("%s (%s)", req.Username, req.School)
			} else {
				profName = req.School
			}
		}

		p := &db.Profile{
			ID:       profID,
			Name:     profName,
			School:   req.School,
			Server:   req.Server,
			Username: req.Username,
			Password: req.Password,
			IsActive: req.SetActive,
		}

		if err := s.database.SaveProfile(p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		if req.SetActive {
			_ = s.database.SetActiveProfile(profID)
			s.mu.Lock()
			s.activeClient = testClient
			s.mu.Unlock()
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"profileId":   profID,
			"displayName": testClient.UserInfo.DisplayName,
			"message":     fmt.Sprintf("Erfolgreich angemeldet als %s", testClient.UserInfo.DisplayName),
		})

	case http.MethodDelete:
		profID := r.URL.Query().Get("id")
		if profID == "" {
			var req struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			profID = req.ID
		}

		if profID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Profil-ID fehlt"})
			return
		}

		if err := s.database.DeleteProfile(profID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		// Recheck active profile
		activeProf, _ := s.database.GetActiveProfile()
		s.mu.Lock()
		if activeProf != nil {
			pwd, _ := s.database.GetDecryptedPassword(activeProf)
			s.activeClient = api.NewClient(activeProf.Server, activeProf.School, activeProf.Username, pwd, "password")
			go s.activeClient.Authenticate()
		} else {
			s.activeClient = nil
		}
		s.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler: /api/profiles/switch
func (s *Server) handleProfileSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProfileID string `json:"profileId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProfileID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Ungültige ProfileID"})
		return
	}

	if err := s.database.SetActiveProfile(req.ProfileID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
		return
	}

	activeProf, err := s.database.GetProfile(req.ProfileID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
		return
	}

	pwd, _ := s.database.GetDecryptedPassword(activeProf)
	newClient := api.NewClient(activeProf.Server, activeProf.School, activeProf.Username, pwd, "password")

	// Reset selected class for newly activated school
	_ = s.database.SetIntSetting("selected_class_id", 0)
	_ = s.database.SetSetting("selected_class_name", "")

	s.mu.Lock()
	s.activeClient = newClient
	s.mu.Unlock()

	go func() {
		_ = newClient.Authenticate()
	}()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Zu Profil '%s' gewechselt", activeProf.Name),
		"profile": activeProf,
	})
}

// Handler: /api/profiles/delete
func (s *Server) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profID := r.URL.Query().Get("id")
	if profID == "" {
		var req struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		profID = req.ID
	}

	if profID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Profil-ID fehlt"})
		return
	}

	if err := s.database.DeleteProfile(profID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
		return
	}

	// Recheck active profile
	activeProf, _ := s.database.GetActiveProfile()
	s.mu.Lock()
	if activeProf != nil {
		pwd, _ := s.database.GetDecryptedPassword(activeProf)
		s.activeClient = api.NewClient(activeProf.Server, activeProf.School, activeProf.Username, pwd, "password")
		go s.activeClient.Authenticate()
	} else {
		s.activeClient = nil
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profil erfolgreich gelöscht",
	})
}

// Handler: /api/schools/search
func (s *Server) handleSchoolSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"schools": []api.SchoolSearchResult{}})
		return
	}

	schools, err := api.SearchSchool(q)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"error": err.Error(), "schools": []api.SchoolSearchResult{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"schools": schools})
}

// Handler: /api/classes
func (s *Server) handleClasses(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []db.Class{})
		return
	}

	school := client.School

	// 1. Cache-First: Load classes from SQLite (< 1ms)
	cachedClasses, err := s.database.GetClasses(school)
	if err == nil && len(cachedClasses) > 0 && r.URL.Query().Get("force") != "true" {
		writeJSON(w, http.StatusOK, cachedClasses)
		return
	}

	// 2. Fetch fresh classes from WebUntis API
	apiClasses, err := client.GetKlassen()
	if err != nil {
		// Fallback to cache even if stale
		if len(cachedClasses) > 0 {
			writeJSON(w, http.StatusOK, cachedClasses)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"error": err.Error(), "classes": []db.Class{}})
		return
	}

	var dbClasses []db.Class
	for _, ac := range apiClasses {
		dbClasses = append(dbClasses, db.Class{
			ID:       ac.ID,
			School:   school,
			Name:     ac.Name,
			LongName: ac.LongName,
			Active:   ac.Active,
		})
	}

	// Persist to SQLite
	_ = s.database.SaveClasses(school, dbClasses)

	// If no class selected yet, auto-select detected class or first class
	selID := s.database.GetIntSetting("selected_class_id", 0)
	if selID == 0 && len(dbClasses) > 0 {
		picked := dbClasses[0]
		if client.UserInfo.DetectedClass != "" {
			for _, c := range dbClasses {
				if strings.EqualFold(c.Name, client.UserInfo.DetectedClass) {
					picked = c
					break
				}
			}
		}
		_ = s.database.SetIntSetting("selected_class_id", picked.ID)
		_ = s.database.SetSetting("selected_class_name", picked.Name)
	}

	writeJSON(w, http.StatusOK, dbClasses)
}

// Handler: /api/timetable
func (s *Server) handleTimetable(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	q := r.URL.Query()
	classID := s.database.GetIntSetting("selected_class_id", 0)
	if cIDStr := q.Get("classId"); cIDStr != "" {
		if id, err := strconv.Atoi(cIDStr); err == nil && id > 0 {
			classID = id
		}
	}

	if classID == 0 {
		// Attempt to pick first class from database
		classes, _ := s.database.GetClasses(client.School)
		if len(classes) > 0 {
			classID = classes[0].ID
			_ = s.database.SetIntSetting("selected_class_id", classID)
			_ = s.database.SetSetting("selected_class_name", classes[0].Name)
		} else {
			writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
			return
		}
	}

	dateStr := q.Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = t
		}
	}

	view := q.Get("view")
	if view == "" {
		view = s.database.GetSetting("default_view", "day")
	}

	var startDate, endDate time.Time
	if view == "week" {
		weekday := targetDate.Weekday()
		diffToMonday := int(time.Monday - weekday)
		if weekday == time.Sunday {
			diffToMonday = -6
		}
		monday := targetDate.AddDate(0, 0, diffToMonday)
		startDate = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, targetDate.Location())
		friday := monday.AddDate(0, 0, 4)
		endDate = time.Date(friday.Year(), friday.Month(), friday.Day(), 23, 59, 59, 0, targetDate.Location())
	} else {
		startDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
		endDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location())
	}

	dateKey := startDate.Format("2006-01-02")
	forceRefresh := q.Get("force") == "true"

	// ZERO-LAG CACHE-FIRST:
	// If cached in SQLite, deliver in < 1ms!
	cachedJSON, updatedAt, found, err := s.database.GetTimetableCache(classID, dateKey)
	if err == nil && found && !forceRefresh && cachedJSON != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Cache-Lookup", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cachedJSON))

		// Asynchronously refresh cache in background if older than 15 minutes
		if time.Since(updatedAt) > 15*time.Minute {
			go func() {
				s.fetchAndCacheTimetable(client, classID, startDate, endDate, dateKey)
			}()
		}
		return
	}

	// Fetch directly
	lessons, err := s.fetchAndCacheTimetable(client, classID, startDate, endDate, dateKey)
	if err != nil {
		log.Printf("[Timetable] Fehler beim Abrufen für Klasse %d: %v", classID, err)
		if found && cachedJSON != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cachedJSON))
			return
		}
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	writeJSON(w, http.StatusOK, lessons)
}

func (s *Server) fetchAndCacheTimetable(client *api.Client, classID int, startDate, endDate time.Time, dateKey string) ([]api.EnrichedLesson, error) {
	lessons, err := client.GetTimetable(classID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(lessons) > 0 {
		if dataBytes, err := json.Marshal(lessons); err == nil {
			_ = s.database.SaveTimetableCache(classID, dateKey, string(dataBytes))
		}
	}
	return lessons, nil
}

// Handler: /api/dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	activeProf, err := s.database.GetActiveProfile()
	if err != nil || activeProf == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"needsOnboarding": true,
		})
		return
	}

	var displayName string
	if client != nil && client.UserInfo.DisplayName != "" {
		displayName = client.UserInfo.DisplayName
	} else {
		displayName = activeProf.Name
	}

	now := time.Now()
	hour := now.Hour()
	greeting := "Guten Tag"
	if hour < 11 {
		greeting = "Guten Morgen"
	} else if hour >= 17 {
		greeting = "Guten Abend"
	}
	greetingFull := fmt.Sprintf("%s, %s", greeting, displayName)

	germanWeekday := api.GermanDayName(now.Weekday())
	dateFormatted := fmt.Sprintf("%s, %d. %s %d", germanWeekday, now.Day(), germanMonthName(now.Month()), now.Year())

	var (
		todayLessons        []api.EnrichedLesson
		nextLesson          *api.EnrichedLesson
		isUpcomingSchoolDay bool
		upcomingDayLabel    string
		openHomework        []db.Homework
		openHwCount         int
		recentMessages      []api.Message
		messagesCount       int
		totalAbs            int
		excAbs              int
		unexcAbs            int
		mu                  sync.Mutex
		wg                  sync.WaitGroup
	)

	// 1. Timetable (Concurrent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if client == nil {
			return
		}
		var lessons []api.EnrichedLesson
		if l, err := client.GetOwnTimetable(now, now); err == nil && len(l) > 0 {
			lessons = l
		} else {
			classID := s.database.GetIntSetting("selected_class_id", 0)
			if classID != 0 {
				if clLessons, errCl := client.GetTimetable(classID, now, now); errCl == nil {
					lessons = clLessons
				}
			}
		}

		nowTimeStr := now.Format("15:04")
		var nextL *api.EnrichedLesson
		for _, l := range lessons {
			if !l.IsCancelled && l.EndTimeStr >= nowTimeStr {
				nextL = &l
				break
			}
		}

		var isNextDay bool
		var nextLabel string
		// If today has no lessons or after 17:00 when school is done, determine next school day
		if len(lessons) == 0 || (now.Hour() >= 17 && nextL == nil) {
			nextDay := now.AddDate(0, 0, 1)
			for nextDay.Weekday() == time.Saturday || nextDay.Weekday() == time.Sunday {
				nextDay = nextDay.AddDate(0, 0, 1)
			}
			if nl, err := client.GetOwnTimetable(nextDay, nextDay); err == nil && len(nl) > 0 {
				lessons = nl
				isNextDay = true
				nextLabel = fmt.Sprintf("Nächster Schultag (%s, %02d.%02d.)", api.GermanDayName(nextDay.Weekday()), nextDay.Day(), int(nextDay.Month()))
				if nextL == nil && len(nl) > 0 {
					nextL = &nl[0]
				}
			}
		}

		mu.Lock()
		todayLessons = lessons
		nextLesson = nextL
		isUpcomingSchoolDay = isNextDay
		upcomingDayLabel = nextLabel
		mu.Unlock()
	}()

	// 2. Homework (Concurrent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var hwList []db.Homework
		var hwCount int
		if localHw, err := s.database.GetHomeworks(activeProf.ID); err == nil {
			for _, h := range localHw {
				if !h.Completed {
					hwCount++
					if len(hwList) < 5 {
						hwList = append(hwList, h)
					}
				}
			}
		}
		if client != nil {
			if wuHw, err := client.GetHomeworks(now.AddDate(0, 0, -7), now.AddDate(0, 0, 14)); err == nil {
				for _, h := range wuHw {
					if !h.Completed {
						hwCount++
						if len(hwList) < 5 {
							hwList = append(hwList, db.Homework{
								ID:          fmt.Sprintf("wu_%d", h.ID),
								ProfileID:   activeProf.ID,
								Subject:     h.Subject,
								Description: h.Text,
								DueDate:     h.DueDateStr,
								Completed:   h.Completed,
								Source:      "webuntis",
							})
						}
					}
				}
			}
		}
		mu.Lock()
		openHomework = hwList
		openHwCount = hwCount
		mu.Unlock()
	}()

	// 3. Messages (Concurrent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if client == nil {
			return
		}
		if msgs, err := client.GetMessages(); err == nil {
			mu.Lock()
			messagesCount = len(msgs)
			if len(msgs) > 5 {
				recentMessages = msgs[:5]
			} else {
				recentMessages = msgs
			}
			mu.Unlock()
		}
	}()

	// 4. Absences (Concurrent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var tot, exc, unexc int
		if localAbs, err := s.database.GetAbsences(activeProf.ID); err == nil {
			tot += len(localAbs)
			for _, a := range localAbs {
				if a.IsExcused {
					exc++
				} else {
					unexc++
				}
			}
		}
		if client != nil {
			startYear := now.Year()
			if now.Month() < time.August {
				startYear--
			}
			schoolYearStart := time.Date(startYear, 8, 1, 0, 0, 0, 0, time.Local)
			schoolYearEnd := time.Date(startYear+1, 7, 31, 23, 59, 59, 0, time.Local)
			if wuAbs, err := client.GetAbsences(schoolYearStart, schoolYearEnd); err == nil {
				tot += len(wuAbs)
				for _, a := range wuAbs {
					if a.IsExcused {
						exc++
					} else {
						unexc++
					}
				}
			}
		}
		mu.Lock()
		totalAbs = tot
		excAbs = exc
		unexcAbs = unexc
		mu.Unlock()
	}()

	wg.Wait()

	if todayLessons == nil {
		todayLessons = []api.EnrichedLesson{}
	}
	if openHomework == nil {
		openHomework = []db.Homework{}
	}
	if recentMessages == nil {
		recentMessages = []api.Message{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needsOnboarding":     false,
		"greeting":            greetingFull,
		"displayName":         displayName,
		"school":              activeProf.School,
		"dateFormatted":       dateFormatted,
		"todayLessons":        todayLessons,
		"nextLesson":          nextLesson,
		"isUpcomingSchoolDay": isUpcomingSchoolDay,
		"upcomingDayLabel":    upcomingDayLabel,
		"openHomeworkCount":   openHwCount,
		"openHomework":        openHomework,
		"messagesCount":       messagesCount,
		"recentMessages":      recentMessages,
		"absencesSummary": map[string]int{
			"total":     totalAbs,
			"excused":   excAbs,
			"unexcused": unexcAbs,
		},
	})
}

// Handler: /api/timetable/own
func (s *Server) handleOwnTimetable(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	q := r.URL.Query()
	dateStr := q.Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = t
		}
	}

	view := q.Get("view")
	if view == "" {
		view = s.database.GetSetting("default_view", "day")
	}

	var startDate, endDate time.Time
	if view == "week" {
		weekday := targetDate.Weekday()
		diffToMonday := int(time.Monday - weekday)
		if weekday == time.Sunday {
			diffToMonday = -6
		}
		monday := targetDate.AddDate(0, 0, diffToMonday)
		startDate = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, targetDate.Location())
		friday := monday.AddDate(0, 0, 4)
		endDate = time.Date(friday.Year(), friday.Month(), friday.Day(), 23, 59, 59, 0, targetDate.Location())
	} else {
		startDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
		endDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location())
	}

	dateKey := startDate.Format("2006-01-02")
	cacheKey := fmt.Sprintf("own_%s_%s", view, dateKey)
	forceRefresh := q.Get("force") == "true"

	// Zero-Lag SQLite Cache Lookup (< 1ms)
	cachedJSON, updatedAt, found, err := s.database.GetTimetableCache(-100, cacheKey)
	if err == nil && found && !forceRefresh && cachedJSON != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Cache-Lookup", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cachedJSON))

		if time.Since(updatedAt) > 15*time.Minute {
			go func() {
				if l, err := client.GetOwnTimetable(startDate, endDate); err == nil && len(l) > 0 {
					if b, errM := json.Marshal(l); errM == nil {
						_ = s.database.SaveTimetableCache(-100, cacheKey, string(b))
					}
				}
			}()
		}
		return
	}

	lessons, err := client.GetOwnTimetable(startDate, endDate)
	if err != nil || lessons == nil {
		if err != nil {
			log.Printf("[OwnTimetable] Fehler: %v", err)
		}
		if found && cachedJSON != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cachedJSON))
			return
		}
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	if len(lessons) > 0 {
		if b, errM := json.Marshal(lessons); errM == nil {
			_ = s.database.SaveTimetableCache(-100, cacheKey, string(b))
		}
	}

	writeJSON(w, http.StatusOK, lessons)
}

// Handler: /api/timetable/resource
func (s *Server) handleResourceTimetable(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	q := r.URL.Query()
	resType := strings.ToUpper(strings.TrimSpace(q.Get("type")))
	if resType == "" {
		resType = "CLASS"
	}

	idStr := q.Get("id")
	resID, err := strconv.Atoi(idStr)
	if err != nil || resID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Ungültige Ressourcen-ID"})
		return
	}

	dateStr := q.Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = t
		}
	}

	view := q.Get("view")
	if view == "" {
		view = s.database.GetSetting("default_view", "day")
	}

	var startDate, endDate time.Time
	if view == "week" {
		weekday := targetDate.Weekday()
		diffToMonday := int(time.Monday - weekday)
		if weekday == time.Sunday {
			diffToMonday = -6
		}
		monday := targetDate.AddDate(0, 0, diffToMonday)
		startDate = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, targetDate.Location())
		friday := monday.AddDate(0, 0, 4)
		endDate = time.Date(friday.Year(), friday.Month(), friday.Day(), 23, 59, 59, 0, targetDate.Location())
	} else {
		startDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
		endDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location())
	}

	if resType == "ROOM" {
		// 1. Try direct WebUntis room query
		lessons, err := client.GetTimetableForResource("ROOM", resID, startDate, endDate, "STANDARD")
		if err == nil && len(lessons) > 0 {
			writeJSON(w, http.StatusOK, lessons)
			return
		}

		// 2. Calculate room timetable from class lessons
		roomLessons := s.computeRoomTimetable(client, resID, startDate, endDate)
		writeJSON(w, http.StatusOK, roomLessons)
		return
	}

	lessons, err := client.GetTimetableForResource(resType, resID, startDate, endDate, "STANDARD")
	if err != nil || lessons == nil {
		if err != nil {
			log.Printf("[ResourceTimetable] Fehler für %s %d: %v", resType, resID, err)
		}
		writeJSON(w, http.StatusOK, []api.EnrichedLesson{})
		return
	}

	writeJSON(w, http.StatusOK, lessons)
}

func (s *Server) computeRoomTimetable(client *api.Client, roomID int, startDate, endDate time.Time) []api.EnrichedLesson {
	rooms, err := client.GetRooms()
	if err != nil || len(rooms) == 0 {
		return []api.EnrichedLesson{}
	}

	var roomName string
	for _, r := range rooms {
		if r.ID == roomID {
			roomName = r.Name
			break
		}
	}

	if roomName == "" {
		return []api.EnrichedLesson{}
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	jsonBlobs, _ := s.database.FindCachedLessonsRange(startStr, endStr)
	if len(jsonBlobs) < 3 {
		classes, _ := s.database.GetClasses(client.School)
		if len(classes) == 0 {
			classesApi, _ := client.GetKlassen()
			for _, ca := range classesApi {
				classes = append(classes, db.Class{ID: ca.ID, School: client.School, Name: ca.Name})
			}
		}
		limit := 8
		if len(classes) < limit {
			limit = len(classes)
		}
		var wg sync.WaitGroup
		for i := 0; i < limit; i++ {
			cID := classes[i].ID
			wg.Add(1)
			go func(clsID int) {
				defer wg.Done()
				_, _ = s.fetchAndCacheTimetable(client, clsID, startDate, endDate, startStr)
			}(cID)
		}
		wg.Wait()
		jsonBlobs, _ = s.database.FindCachedLessonsRange(startStr, endStr)
	}

	type lessonKey struct {
		Date      string
		StartTime string
		Subject   string
		Class     string
	}
	seen := make(map[lessonKey]bool)
	var matched []api.EnrichedLesson

	for _, blob := range jsonBlobs {
		var lessons []api.EnrichedLesson
		if err := json.Unmarshal([]byte(blob), &lessons); err != nil {
			continue
		}
		for _, l := range lessons {
			rClean := strings.TrimSpace(l.Room)
			if strings.EqualFold(rClean, roomName) || strings.Contains(strings.ToLower(rClean), strings.ToLower(roomName)) {
				key := lessonKey{
					Date:      l.Date,
					StartTime: l.StartTimeStr,
					Subject:   l.Subject,
					Class:     l.Class,
				}
				if !seen[key] {
					seen[key] = true
					matched = append(matched, l)
				}
			}
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Date != matched[j].Date {
			return matched[i].Date < matched[j].Date
		}
		return matched[i].StartTimeStr < matched[j].StartTimeStr
	})

	return matched
}

// Handler: /api/teachers
func (s *Server) handleTeachers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.Teacher{})
		return
	}

	teachers, err := client.GetTeachers()
	if err != nil {
		writeJSON(w, http.StatusOK, []api.Teacher{})
		return
	}

	writeJSON(w, http.StatusOK, teachers)
}

// Handler: /api/rooms
func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.Room{})
		return
	}

	rooms, err := client.GetRooms()
	if err != nil {
		writeJSON(w, http.StatusOK, []api.Room{})
		return
	}

	writeJSON(w, http.StatusOK, rooms)
}

// Handler: /api/messages
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusOK, []api.Message{})
		return
	}

	if idStr := r.URL.Query().Get("id"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			msg, err := client.GetMessageById(id)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, msg)
			return
		}
	}

	messages, err := client.GetMessages()
	if err != nil {
		writeJSON(w, http.StatusOK, []api.Message{})
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// Handler: /api/messages/{id}
func (s *Server) handleMessageDetail(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Kein aktives Profil"})
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/messages/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Ungültige Nachrichten-ID"})
		return
	}

	msg, err := client.GetMessageById(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// Handler: /api/homework
func (s *Server) handleHomework(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	activeProf, _ := s.database.GetActiveProfile()
	profID := ""
	if activeProf != nil {
		profID = activeProf.ID
	}

	switch r.Method {
	case http.MethodGet:
		var combined []db.Homework

		// 1. Local SQLite homework
		if localHw, err := s.database.GetHomeworks(profID); err == nil {
			combined = append(combined, localHw...)
		}

		// 2. WebUntis homework
		if client != nil {
			now := time.Now()
			start := now.AddDate(0, -1, 0)
			end := now.AddDate(0, 2, 0)
			if wuHw, err := client.GetHomeworks(start, end); err == nil {
				for _, h := range wuHw {
					combined = append(combined, db.Homework{
						ID:          fmt.Sprintf("wu_%d", h.ID),
						ProfileID:   profID,
						Subject:     h.Subject,
						Description: h.Text,
						DueDate:     h.DueDateStr,
						Completed:   h.Completed,
						Source:      "webuntis",
					})
				}
			}
		}

		sort.Slice(combined, func(i, j int) bool {
			if combined[i].Completed != combined[j].Completed {
				return !combined[i].Completed
			}
			return combined[i].DueDate < combined[j].DueDate
		})

		writeJSON(w, http.StatusOK, combined)

	case http.MethodPost:
		var req struct {
			Subject     string `json:"subject"`
			Description string `json:"description"`
			DueDate     string `json:"dueDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Ungültiger Anfragekörper"})
			return
		}

		if req.Description == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Beschreibung ist erforderlich"})
			return
		}

		if req.DueDate == "" {
			req.DueDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		}

		h := &db.Homework{
			ProfileID:   profID,
			Subject:     req.Subject,
			Description: req.Description,
			DueDate:     req.DueDate,
			Completed:   false,
		}

		if err := s.database.CreateHomework(h); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "homework": h})

	case http.MethodPut:
		var req struct {
			ID        string `json:"id"`
			Completed bool   `json:"completed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "ID fehlt"})
			return
		}

		if strings.HasPrefix(req.ID, "wu_") {
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
			return
		}

		if err := s.database.UpdateHomeworkCompleted(req.ID, req.Completed); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			var req struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			id = req.ID
		}

		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "ID fehlt"})
			return
		}

		if err := s.database.DeleteHomework(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler: /api/absences
func (s *Server) handleAbsences(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	activeProf, _ := s.database.GetActiveProfile()
	profID := ""
	if activeProf != nil {
		profID = activeProf.ID
	}

	switch r.Method {
	case http.MethodGet:
		var combined []db.Absence

		yearParam := r.URL.Query().Get("year")
		now := time.Now()
		startYear := now.Year()
		if now.Month() < time.August {
			startYear--
		}
		currentSchoolYear := fmt.Sprintf("%d/%d", startYear, startYear+1)
		if yearParam == "" {
			yearParam = currentSchoolYear
		}

		var filterStart, filterEnd string
		if yearParam != "all" {
			parts := strings.Split(yearParam, "/")
			if len(parts) == 2 {
				if sy, err := strconv.Atoi(parts[0]); err == nil {
					filterStart = fmt.Sprintf("%04d-08-01", sy)
					filterEnd = fmt.Sprintf("%04d-07-31", sy+1)
				}
			}
		}

		// 1. Local SQLite absences
		if localAbs, err := s.database.GetAbsences(profID); err == nil {
			for _, a := range localAbs {
				if filterStart != "" && (a.StartDate < filterStart || a.StartDate > filterEnd) {
					continue
				}
				combined = append(combined, a)
			}
		}

		// 2. WebUntis absences
		if client != nil {
			var queryStart, queryEnd time.Time
			if filterStart != "" {
				queryStart, _ = time.Parse("2006-01-02", filterStart)
				queryEnd, _ = time.Parse("2006-01-02", filterEnd)
			} else {
				queryStart = now.AddDate(-2, 0, 0)
				queryEnd = now.AddDate(1, 0, 0)
			}

			if wuAbs, err := client.GetAbsences(queryStart, queryEnd); err == nil {
				for _, a := range wuAbs {
					if filterStart != "" && (a.StartDateStr < filterStart || a.StartDateStr > filterEnd) {
						continue
					}
					combined = append(combined, db.Absence{
						ID:        fmt.Sprintf("wu_%d", a.ID),
						ProfileID: profID,
						Reason:    a.Reason,
						Text:      a.Text,
						StartDate: a.StartDateStr,
						EndDate:   a.EndDateStr,
						IsExcused: a.IsExcused,
						Source:    "webuntis",
					})
				}
			}
		}

		sort.Slice(combined, func(i, j int) bool {
			return combined[i].StartDate > combined[j].StartDate
		})

		w.Header().Set("X-Selected-School-Year", yearParam)
		w.Header().Set("X-Current-School-Year", currentSchoolYear)

		if r.URL.Query().Get("format") == "envelope" || r.URL.Query().Get("format") == "full" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"absences":           combined,
				"selectedSchoolYear": yearParam,
				"currentSchoolYear":  currentSchoolYear,
			})
			return
		}

		writeJSON(w, http.StatusOK, combined)

	case http.MethodPost:
		var req struct {
			Reason    string `json:"reason"`
			Text      string `json:"text"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
			IsExcused bool   `json:"isExcused"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Ungültiger Anfragekörper"})
			return
		}

		if req.Reason == "" {
			req.Reason = "Krankmeldung"
		}

		if req.StartDate == "" {
			req.StartDate = time.Now().Format("2006-01-02")
		}
		if req.EndDate == "" {
			req.EndDate = req.StartDate
		}

		a := &db.Absence{
			ProfileID: profID,
			Reason:    req.Reason,
			Text:      req.Text,
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			IsExcused: req.IsExcused,
		}

		if err := s.database.CreateAbsence(a); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "absence": a})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			var req struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			id = req.ID
		}

		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "ID fehlt"})
			return
		}

		if err := s.database.DeleteAbsence(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func germanMonthName(m time.Month) string {
	switch m {
	case time.January:
		return "Januar"
	case time.February:
		return "Februar"
	case time.March:
		return "März"
	case time.April:
		return "April"
	case time.May:
		return "Mai"
	case time.June:
		return "Juni"
	case time.July:
		return "Juli"
	case time.August:
		return "August"
	case time.September:
		return "September"
	case time.October:
		return "Oktober"
	case time.November:
		return "November"
	case time.December:
		return "Dezember"
	default:
		return ""
	}
}

// Handler: /api/settings
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := s.database.GetAllSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, settings)

	case http.MethodPost:
		var incoming map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}

		for k, v := range incoming {
			switch val := v.(type) {
			case string:
				_ = s.database.SetSetting(k, val)
			case float64:
				_ = s.database.SetIntSetting(k, int(val))
			case int:
				_ = s.database.SetIntSetting(k, val)
			case bool:
				_ = s.database.SetSetting(k, strconv.FormatBool(val))
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler: /api/settings/aliases
func (s *Server) handleSubjectAliases(w http.ResponseWriter, r *http.Request) {
	activeProf, err := s.database.GetActiveProfile()
	if err != nil || activeProf == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Kein aktives Profil"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		aliases, err := s.database.GetSubjectAliases(activeProf.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, aliases)

	case http.MethodPost:
		var req struct {
			Original string `json:"original"`
			Alias    string `json:"alias"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Original) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Ungültige Eingabe"})
			return
		}
		if err := db.ValidateCustomAlias(req.Alias); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		if err := s.database.SetSubjectAlias(activeProf.ID, strings.TrimSpace(req.Original), strings.TrimSpace(req.Alias)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	case http.MethodDelete:
		orig := r.URL.Query().Get("original")
		if orig == "" {
			var req struct {
				Original string `json:"original"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			orig = req.Original
		}
		if orig == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Original-Fach fehlt"})
			return
		}
		if err := s.database.DeleteSubjectAlias(activeProf.ID, orig); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler: /api/refresh
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	client := s.activeClient
	s.mu.RUnlock()

	if client != nil {
		_ = s.database.ClearTimetableCache()
		go func() {
			_ = client.Authenticate()
		}()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// Handler: static files
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/static/")
	data, err := web.Assets.ReadFile(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case strings.HasSuffix(filename, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(filename, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(filename, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(filename, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// Handler: index.html
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := web.Assets.ReadFile("index.html")
	if err != nil {
		http.Error(w, "index.html missing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Handler: /api/updates/check
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	info, err := updater.CheckForUpdate(AppVersion)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"currentVersion": AppVersion,
			"hasUpdate":      false,
			"error":          err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// Handler: /api/updates/apply
func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DownloadURL string `json:"downloadUrl"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	downloadURL := req.DownloadURL
	if downloadURL == "" {
		info, err := updater.CheckForUpdate(AppVersion)
		if err != nil || info.DownloadURL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Keine Download-URL für dieses System gefunden",
			})
			return
		}
		downloadURL = info.DownloadURL
	}

	if err := updater.ApplyUpdate(downloadURL); err != nil {
		log.Printf("[Update] Fehler beim Aktualisieren: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Update fehlgeschlagen: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Update erfolgreich installiert! Bitte starte die Anwendung neu.",
	})
}

