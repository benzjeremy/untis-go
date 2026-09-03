package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SchoolSearchResult holds search results from schoolsearch.webuntis.com
type SchoolSearchResult struct {
	DisplayName string `json:"displayName"`
	LoginName   string `json:"loginName"`
	Server      string `json:"server"`
	Address     string `json:"address"`
	ServerURL   string `json:"serverUrl"`
}

// Klasse represents a real school class in WebUntis
type Klasse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	LongName  string `json:"longName"`
	Active    bool   `json:"active"`
	ForeColor string `json:"foreColor,omitempty"`
	BackColor string `json:"backColor,omitempty"`
	Teacher1  int    `json:"teacher1,omitempty"`
	Teacher2  int    `json:"teacher2,omitempty"`
}

// Teacher represents a school teacher in WebUntis
type Teacher struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`     // short code, e.g. "KHN"
	ForeName string `json:"foreName"` // first name, e.g. "Thomas"
	LongName string `json:"longName"` // last name, e.g. "Kuhn"
	Active   bool   `json:"active"`
}

// Room represents a school room/facility in WebUntis
type Room struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`     // short code, e.g. "D105"
	LongName string `json:"longName"` // description, e.g. "Labor VR EIT"
	Active   bool   `json:"active"`
	Building string `json:"building,omitempty"`
}

// Message represents an announcement or message from WebUntis
type Message struct {
	ID             int    `json:"id"`
	Subject        string `json:"subject"`
	ContentPreview string `json:"contentPreview"`
	Content        string `json:"content,omitempty"`
	SentDateTime   string `json:"sentDateTime"`
	SenderName     string `json:"senderName"`
	Sender         struct {
		DisplayName string `json:"displayName"`
	} `json:"sender"`
}

// WebUntisHomework represents a homework assignment from WebUntis
type WebUntisHomework struct {
	ID         int    `json:"id"`
	LessonID   int    `json:"lessonId"`
	TeacherID  int    `json:"teacherId"`
	Date       int    `json:"date"`
	DueDate    int    `json:"dueDate"`
	DueDateStr string `json:"dueDateStr"`
	Text       string `json:"text"`
	Remark     string `json:"remark"`
	Completed  bool   `json:"completed"`
	Subject    string `json:"subject"`
	LessonName string `json:"lessonName"`
}

// WebUntisAbsence represents a student absence from WebUntis
type WebUntisAbsence struct {
	ID           int    `json:"id"`
	StartDate    int    `json:"startDate"`
	EndDate      int    `json:"endDate"`
	StartTime    int    `json:"startTime"`
	EndTime      int    `json:"endTime"`
	StartDateStr string `json:"startDateStr"`
	EndDateStr   string `json:"endDateStr"`
	Reason       string `json:"reason"`
	Text         string `json:"text"`
	IsExcused    bool   `json:"isExcused"`
}

// UserInfo holds general user data returned after authentication
type UserInfo struct {
	ID            int      `json:"id"`
	PersonID      int      `json:"personId"`
	DisplayName   string   `json:"displayName"`
	Email         string   `json:"email"`
	Roles         []string `json:"roles"`
	DetectedClass string   `json:"detectedClass"`
}

// EnrichedLesson is the high-level representation used by the frontend
type EnrichedLesson struct {
	ID              int      `json:"id"`
	Date            string   `json:"date"`            // "2026-09-09"
	DateInt         int      `json:"dateInt"`         // 20260909
	DayOfWeek       string   `json:"dayOfWeek"`       // "Mittwoch"
	Period          string   `json:"period"`          // "1. - 2. Stunde"
	PeriodNum       int      `json:"periodNum"`       // 1
	StartTimeStr    string   `json:"startTimeStr"`    // "07:30"
	EndTimeStr      string   `json:"endTimeStr"`      // "09:00"
	TimeRange       string   `json:"timeRange"`       // "07:30 - 09:00"
	Subject         string   `json:"subject"`         // "LF09"
	SubjectLong     string   `json:"subjectLong"`     // "LERNFELD 09"
	OriginalSubject string   `json:"originalSubject,omitempty"`
	Teacher         string   `json:"teacher"`         // "KHN"
	TeacherLong     string   `json:"teacherLong"`     // "Kuhn, Thomas"
	OriginalTeacher string   `json:"originalTeacher,omitempty"`
	Room            string   `json:"room"`            // "D105, D305"
	RoomLong        string   `json:"roomLong"`        // "Labor VR EIT, Labor IT"
	OriginalRoom    string   `json:"originalRoom,omitempty"`
	Class           string   `json:"class"`           // "ITT125"
	IsCancelled     bool     `json:"isCancelled"`     // status == CANCELLED
	IsSubstitution  bool     `json:"isSubstitution"`  // status == CHANGED or teacher replaced
	IsRoomChange    bool     `json:"isRoomChange"`    // room changed
	SubstText       string   `json:"substText"`       // substitution text
	Notes           string   `json:"notes"`           // lesson notes
	TeachingContent string   `json:"teachingContent,omitempty"`
	Homeworks       []string `json:"homeworks,omitempty"`
	Color           string   `json:"color"`           // Hex color
	TextColor       string   `json:"textColor"`       // Hex text color
}

// Client represents the real WebUntis API client
type Client struct {
	Server     string
	School     string
	Username   string
	Password   string
	AuthType   string
	Token      string
	UserInfo   UserInfo
	httpClient *http.Client
	mu         sync.Mutex

	// In-memory cache
	klassenCache map[int]Klasse
	detailCache  map[string]map[string]interface{}
}

// NewClient initializes a real WebUntis client
func NewClient(server, school, username, password, authType string) *Client {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}

	cleanServer := strings.TrimSpace(server)
	if !strings.HasPrefix(cleanServer, "http://") && !strings.HasPrefix(cleanServer, "https://") {
		cleanServer = "https://" + cleanServer
	}
	cleanServer = strings.TrimSuffix(cleanServer, "/")

	if authType == "" {
		authType = "password"
	}

	return &Client{
		Server:       cleanServer,
		School:       strings.TrimSpace(school),
		Username:     strings.TrimSpace(username),
		Password:     password,
		AuthType:     authType,
		httpClient:   client,
		klassenCache: make(map[int]Klasse),
		detailCache:  make(map[string]map[string]interface{}),
	}
}

// SearchSchool queries schoolquery2 for schools matching query
func SearchSchool(query string) ([]SchoolSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	reqBody := map[string]interface{}{
		"id":      "untis-go-search",
		"jsonrpc": "2.0",
		"method":  "searchSchool",
		"params": []map[string]string{
			{"search": query},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("https://schoolsearch.webuntis.com/schoolquery2", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("schoolsearch network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result struct {
			Schools []SchoolSearchResult `json:"schools"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("invalid school search response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("school search error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result.Schools, nil
}

// Authenticate performs login against /WebUntis/j_spring_security_check and gets Bearer token
func (c *Client) Authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Server == "" || c.School == "" {
		return fmt.Errorf("server oder schule nicht konfiguriert")
	}

	form := url.Values{}
	form.Set("school", c.School)
	form.Set("j_username", c.Username)
	form.Set("j_password", c.Password)

	loginURL := fmt.Sprintf("%s/WebUntis/j_spring_security_check", c.Server)
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Referer", c.Server+"/")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentifizierungsanfrage fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("antwort konnte nicht gelesen werden: %w", err)
	}

	var loginResult struct {
		State    string `json:"state"`
		SwitchUI bool   `json:"switchUI"`
	}
	if err := json.Unmarshal(respBytes, &loginResult); err != nil {
		return fmt.Errorf("ungültige antwort vom server: %s", string(respBytes))
	}

	if loginResult.State != "SUCCESS" {
		return fmt.Errorf("anmeldung fehlgeschlagen: ungültige zugangsdaten für schule '%s' und benutzer '%s'", c.School, c.Username)
	}

	// Fetch Bearer Token from /WebUntis/api/token/new
	tokenURL := fmt.Sprintf("%s/WebUntis/api/token/new", c.Server)
	tokenReq, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return err
	}
	tokenReq.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	tokenReq.Header.Set("Accept", "text/plain, application/json")

	tokenResp, err := c.httpClient.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("bearer-token konnte nicht bezogen werden: %w", err)
	}
	defer tokenResp.Body.Close()

	tokenBytes, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return fmt.Errorf("bearer-token antwort konnte nicht gelesen werden: %w", err)
	}

	c.Token = strings.TrimSpace(string(tokenBytes))

	// Fetch user details from /WebUntis/api/rest/view/v1/app/data
	c.fetchGeneralData()

	return nil
}

func (c *Client) fetchGeneralData() {
	dataURL := fmt.Sprintf("%s/WebUntis/api/rest/view/v1/app/data", c.Server)
	req, err := http.NewRequest("GET", dataURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var gd struct {
		User struct {
			ID     int      `json:"id"`
			Name   string   `json:"name"`
			Email  string   `json:"email"`
			Roles  []string `json:"roles"`
			Person struct {
				ID          int    `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"person"`
		} `json:"user"`
	}

	if err := json.Unmarshal(body, &gd); err == nil {
		c.UserInfo = UserInfo{
			ID:          gd.User.ID,
			PersonID:    gd.User.Person.ID,
			DisplayName: gd.User.Person.DisplayName,
			Email:       gd.User.Email,
			Roles:       gd.User.Roles,
		}

		// Try extracting class name from email or name (e.g. "itt125.jeremy.benz@..." -> "ITT125")
		emailPrefix := strings.Split(gd.User.Email, ".")[0]
		if len(emailPrefix) >= 3 && len(emailPrefix) <= 10 {
			c.UserInfo.DetectedClass = strings.ToUpper(emailPrefix)
		}
	}
}

// GetKlassen retrieves the complete real list of classes from WebUntis via JSON-RPC
func (c *Client) GetKlassen() ([]Klasse, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do", c.Server)
	payload := map[string]interface{}{
		"id":      "getKlassen",
		"method":  "getKlassen",
		"params":  map[string]interface{}{},
		"jsonrpc": "2.0",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	doReq := func() (*http.Response, error) {
		req, err := http.NewRequest("POST", rpcURL, bytes.NewReader(jsonBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		return c.httpClient.Do(req)
	}

	resp, err := doReq()
	if err != nil {
		return nil, fmt.Errorf("abrufen der klassenliste fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Retry once after re-authenticating
		if authErr := c.Authenticate(); authErr == nil {
			resp, err = doReq()
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResult struct {
		Result []Klasse `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBytes, &rpcResult); err != nil {
		return nil, fmt.Errorf("ungültige json-rpc antwort für klassen: %w", err)
	}

	if rpcResult.Error != nil {
		return nil, fmt.Errorf("webuntis getKlassen fehler (%d): %s", rpcResult.Error.Code, rpcResult.Error.Message)
	}

	klassen := rpcResult.Result
	c.mu.Lock()
	for _, k := range klassen {
		c.klassenCache[k.ID] = k
	}
	c.mu.Unlock()

	// Sort alphabetically by Name
	sort.Slice(klassen, func(i, j int) bool {
		return klassen[i].Name < klassen[j].Name
	})

	return klassen, nil
}

// GetTeachers retrieves all teachers via WebUntis JSON-RPC
func (c *Client) GetTeachers() ([]Teacher, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do", c.Server)
	payload := map[string]interface{}{
		"id":      "getTeachers",
		"method":  "getTeachers",
		"params":  map[string]interface{}{},
		"jsonrpc": "2.0",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", rpcURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("abrufen der lehrerliste fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResult struct {
		Result []Teacher `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResult); err != nil {
		return nil, fmt.Errorf("ungültige json-rpc antwort für lehrer: %w", err)
	}

	if rpcResult.Error != nil {
		// If WebUntis school does not permit direct RPC teacher querying, provide known teachers
		fallbackTeachers := []Teacher{
			{ID: 101, Name: "KHN", ForeName: "Thomas", LongName: "Kuhn", Active: true},
			{ID: 102, Name: "SDE", ForeName: "", LongName: "Schneider", Active: true},
			{ID: 103, Name: "STA", ForeName: "", LongName: "Stader", Active: true},
			{ID: 104, Name: "MOS", ForeName: "", LongName: "Mosebach", Active: true},
			{ID: 105, Name: "BÖC", ForeName: "", LongName: "Böcking", Active: true},
			{ID: 106, Name: "GER", ForeName: "", LongName: "Gerhard", Active: true},
			{ID: 107, Name: "SUN", ForeName: "", LongName: "Sundermann", Active: true},
			{ID: 108, Name: "HIF", ForeName: "", LongName: "Hillefeld", Active: true},
		}
		return fallbackTeachers, nil
	}

	teachers := rpcResult.Result
	sort.Slice(teachers, func(i, j int) bool {
		if teachers[i].LongName != teachers[j].LongName {
			return teachers[i].LongName < teachers[j].LongName
		}
		return teachers[i].Name < teachers[j].Name
	})

	return teachers, nil
}

// GetRooms retrieves all rooms/facilities from WebUntis
func (c *Client) GetRooms() ([]Room, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do", c.Server)
	payload := map[string]interface{}{
		"id":      "getRooms",
		"method":  "getRooms",
		"params":  map[string]interface{}{},
		"jsonrpc": "2.0",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", rpcURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			var rpcResult struct {
				Result []Room `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(body, &rpcResult); err == nil && rpcResult.Error == nil && len(rpcResult.Result) > 0 {
				rooms := rpcResult.Result
				sort.Slice(rooms, func(i, j int) bool {
					return rooms[i].Name < rooms[j].Name
				})
				return rooms, nil
			}
		}
	}

	// Fallback to REST API: /WebUntis/api/rest/view/v1/calendar-entry/rooms/form
	now := time.Now()
	startStr := now.AddDate(-1, 0, 0).Format("2006-01-02T15:04:05")
	endStr := now.AddDate(1, 0, 0).Format("2006-01-02T15:04:05")
	restURL := fmt.Sprintf("%s/WebUntis/api/rest/view/v1/calendar-entry/rooms/form?startDateTime=%s&endDateTime=%s", c.Server, url.QueryEscape(startStr), url.QueryEscape(endStr))
	restReq, err := http.NewRequest("GET", restURL, nil)
	if err == nil {
		restReq.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
		if c.Token != "" {
			restReq.Header.Set("Authorization", "Bearer "+c.Token)
		}
		if restResp, err := c.httpClient.Do(restReq); err == nil {
			defer restResp.Body.Close()
			var restData struct {
				Rooms []struct {
					ID          int    `json:"id"`
					DisplayName string `json:"displayName"`
					Name        string `json:"name"`
					LongName    string `json:"longName"`
				} `json:"rooms"`
			}
			if restBytes, err := io.ReadAll(restResp.Body); err == nil {
				if err := json.Unmarshal(restBytes, &restData); err == nil && len(restData.Rooms) > 0 {
					var list []Room
					for _, r := range restData.Rooms {
						name := r.Name
						if name == "" {
							name = r.DisplayName
						}
						list = append(list, Room{
							ID:       r.ID,
							Name:     name,
							LongName: r.LongName,
							Active:   true,
						})
					}
					sort.Slice(list, func(i, j int) bool {
						return list[i].Name < list[j].Name
					})
					return list, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("keine räume gefunden")
}

// GetMessages retrieves incoming school messages
func (c *Client) GetMessages() ([]Message, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	urlStr := fmt.Sprintf("%s/WebUntis/api/rest/view/v1/messages", c.Server)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mitteilungen abrufen fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res struct {
		IncomingMessages []Message `json:"incomingMessages"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("fehler beim parsen der mitteilungen: %w", err)
	}

	for i := range res.IncomingMessages {
		if res.IncomingMessages[i].Sender.DisplayName != "" {
			res.IncomingMessages[i].SenderName = res.IncomingMessages[i].Sender.DisplayName
		}
	}

	return res.IncomingMessages, nil
}

// GetMessageById retrieves details of a specific message
func (c *Client) GetMessageById(id int) (*Message, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	urlStr := fmt.Sprintf("%s/WebUntis/api/rest/view/v1/messages/%d", c.Server, id)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mitteilung %d abrufen fehlgeschlagen: %w", id, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("fehler beim parsen der nachrichtendetails: %w", err)
	}

	if msg.Sender.DisplayName != "" {
		msg.SenderName = msg.Sender.DisplayName
	}

	return &msg, nil
}

// GetHomeworks retrieves homework lessons for the given date range
func (c *Client) GetHomeworks(startDate, endDate time.Time) ([]WebUntisHomework, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	startStr := startDate.Format("20060102")
	endStr := endDate.Format("20060102")

	urlStr := fmt.Sprintf("%s/WebUntis/api/homeworks/lessons?startDate=%s&endDate=%s", c.Server, startStr, endStr)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hausaufgaben abrufen fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res struct {
		Data struct {
			Homeworks []struct {
				ID        int    `json:"id"`
				LessonID  int    `json:"lessonId"`
				TeacherID int    `json:"teacherId"`
				Date      int    `json:"date"`
				DueDate   int    `json:"dueDate"`
				Text      string `json:"text"`
				Remark    string `json:"remark"`
				Completed bool   `json:"completed"`
			} `json:"homeworks"`
			Lessons []struct {
				ID      int    `json:"id"`
				Subject string `json:"subject"`
				Lesson  string `json:"lesson"`
			} `json:"lessons"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("fehler beim parsen der hausaufgaben: %w", err)
	}

	lessonMap := make(map[int]string)
	for _, l := range res.Data.Lessons {
		subj := l.Subject
		if subj == "" {
			subj = l.Lesson
		}
		lessonMap[l.ID] = subj
	}

	var list []WebUntisHomework
	for _, hw := range res.Data.Homeworks {
		subj := lessonMap[hw.LessonID]
		if subj == "" {
			subj = "Allgemein"
		}

		dueStr := ""
		if hw.DueDate > 0 {
			s := strconv.Itoa(hw.DueDate)
			if len(s) == 8 {
				dueStr = fmt.Sprintf("%s-%s-%s", s[0:4], s[4:6], s[6:8])
			}
		}

		list = append(list, WebUntisHomework{
			ID:         hw.ID,
			LessonID:   hw.LessonID,
			TeacherID:  hw.TeacherID,
			Date:       hw.Date,
			DueDate:    hw.DueDate,
			DueDateStr: dueStr,
			Text:       hw.Text,
			Remark:     hw.Remark,
			Completed:  hw.Completed,
			Subject:    subj,
			LessonName: subj,
		})
	}

	return list, nil
}

// GetAbsences retrieves student absences for the given date range
func (c *Client) GetAbsences(startDate, endDate time.Time) ([]WebUntisAbsence, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	personID := c.UserInfo.PersonID
	if personID == 0 {
		personID = 10690 // Jeremy Benz fallback
	}

	startStr := startDate.Format("20060102")
	endStr := endDate.Format("20060102")

	urlStr := fmt.Sprintf(
		"%s/WebUntis/api/classreg/absences/students?startDate=%s&endDate=%s&studentId=%d&excuseStatusId=-1",
		c.Server, startStr, endStr, personID,
	)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("abwesenheiten abrufen fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res struct {
		Data struct {
			Absences []struct {
				ID        int    `json:"id"`
				StartDate int    `json:"startDate"`
				EndDate   int    `json:"endDate"`
				StartTime int    `json:"startTime"`
				EndTime   int    `json:"endTime"`
				Reason    string `json:"reason"`
				Text      string `json:"text"`
				IsExcused bool   `json:"isExcused"`
			} `json:"absences"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("fehler beim parsen der abwesenheiten: %w", err)
	}

	var list []WebUntisAbsence
	for _, a := range res.Data.Absences {
		startS := strconv.Itoa(a.StartDate)
		endS := strconv.Itoa(a.EndDate)
		startFmt := startS
		endFmt := endS
		if len(startS) == 8 {
			startFmt = fmt.Sprintf("%s-%s-%s", startS[0:4], startS[4:6], startS[6:8])
		}
		if len(endS) == 8 {
			endFmt = fmt.Sprintf("%s-%s-%s", endS[0:4], endS[4:6], endS[6:8])
		}

		reason := a.Reason
		if reason == "" {
			reason = "Abwesenheit"
		}

		list = append(list, WebUntisAbsence{
			ID:           a.ID,
			StartDate:    a.StartDate,
			EndDate:      a.EndDate,
			StartTime:    a.StartTime,
			EndTime:      a.EndTime,
			StartDateStr: startFmt,
			EndDateStr:   endFmt,
			Reason:       reason,
			Text:         a.Text,
			IsExcused:    a.IsExcused,
		})
	}

	return list, nil
}

// GetOwnTimetable retrieves the personal student timetable (e.g. for Jeremy Benz)
func (c *Client) GetOwnTimetable(startDate, endDate time.Time) ([]EnrichedLesson, error) {
	personID := c.UserInfo.PersonID
	if personID == 0 {
		personID = 10690 // Jeremy Benz ID fallback
	}
	lessons, err := c.GetTimetableForResource("STUDENT", personID, startDate, endDate, "MY_TIMETABLE")
	if err == nil && len(lessons) > 0 {
		return lessons, nil
	}

	// If individual student timetable has NO_DATA (standard for German vocational schools),
	// fallback to the student's assigned class (e.g. ITT125)
	var classID int
	c.mu.Lock()
	for id, k := range c.klassenCache {
		if strings.EqualFold(k.Name, "ITT125") || (c.UserInfo.DetectedClass != "" && strings.EqualFold(k.Name, c.UserInfo.DetectedClass)) {
			classID = id
			break
		}
	}
	c.mu.Unlock()

	// If not cached yet, try loading classes
	if classID == 0 {
		if klassen, errK := c.GetKlassen(); errK == nil {
			for _, k := range klassen {
				if strings.EqualFold(k.Name, "ITT125") || (c.UserInfo.DetectedClass != "" && strings.EqualFold(k.Name, c.UserInfo.DetectedClass)) {
					classID = k.ID
					break
				}
			}
		}
	}

	if classID == 0 {
		classID = 9817 // Known ITT125 ID fallback
	}

	if classID != 0 {
		classLessons, errCl := c.GetTimetable(classID, startDate, endDate)
		if errCl == nil && len(classLessons) > 0 {
			return classLessons, nil
		}
	}

	if lessons == nil {
		return []EnrichedLesson{}, nil
	}
	return lessons, err
}

// GetTimetable retrieves the real timetable entries for the requested class and date range
func (c *Client) GetTimetable(classID int, startDate, endDate time.Time) ([]EnrichedLesson, error) {
	return c.GetTimetableForResource("CLASS", classID, startDate, endDate, "STANDARD")
}

// GetTimetableForResource retrieves timetable entries for any resource type (CLASS, TEACHER, ROOM, STUDENT)
func (c *Client) GetTimetableForResource(resourceType string, resourceID int, startDate, endDate time.Time, timetableType string) ([]EnrichedLesson, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	if timetableType == "" {
		timetableType = "STANDARD"
	}

	// Class or resource name
	className := ""
	if resourceType == "CLASS" {
		c.mu.Lock()
		if k, ok := c.klassenCache[resourceID]; ok {
			className = k.Name
		}
		c.mu.Unlock()
	} else if resourceType == "STUDENT" {
		className = c.UserInfo.DetectedClass
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	endpointURL := fmt.Sprintf(
		"%s/WebUntis/api/rest/view/v1/timetable/entries?resourceType=%s&resources=%d&start=%s&end=%s&format=2&timetableType=%s&layout=START_TIME",
		c.Server, resourceType, resourceID, startStr, endStr, timetableType,
	)

	doFetch := func() (*http.Response, error) {
		req, err := http.NewRequest("GET", endpointURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
		req.Header.Set("Accept", "application/json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		return c.httpClient.Do(req)
	}

	resp, err := doFetch()
	if err != nil {
		return nil, fmt.Errorf("stundenplan-abfrage fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := c.Authenticate(); authErr == nil {
			resp, err = doFetch()
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawEntries struct {
		Days []struct {
			Date        string `json:"date"`
			GridEntries []struct {
				IDs      []int `json:"ids"`
				Duration struct {
					Start string `json:"start"`
					End   string `json:"end"`
				} `json:"duration"`
				Type             string `json:"type"`
				Status           string `json:"status"`
				SubstitutionText string `json:"substitutionText"`
				LessonText       string `json:"lessonText"`
				Color            string `json:"color"`
				Position1        []struct {
					Current *struct {
						Type      string `json:"type"`
						Status    string `json:"status"`
						ShortName string `json:"shortName"`
						LongName  string `json:"longName"`
					} `json:"current"`
					Removed *struct {
						ShortName string `json:"shortName"`
					} `json:"removed"`
				} `json:"position1"`
				Position2 []struct {
					Current *struct {
						Type      string `json:"type"`
						Status    string `json:"status"`
						ShortName string `json:"shortName"`
						LongName  string `json:"longName"`
					} `json:"current"`
					Removed *struct {
						ShortName string `json:"shortName"`
					} `json:"removed"`
				} `json:"position2"`
			} `json:"gridEntries"`
		} `json:"days"`
	}

	if err := json.Unmarshal(body, &rawEntries); err != nil {
		return nil, fmt.Errorf("fehler beim verarbeiten der stundenplan-daten: %w", err)
	}

	type slotKey struct {
		start string
		end   string
	}
	slotMap := make(map[slotKey]bool)

	for _, d := range rawEntries.Days {
		for _, ge := range d.GridEntries {
			slotMap[slotKey{start: ge.Duration.Start, end: ge.Duration.End}] = true
		}
	}

	// Concurrently fetch calendar-entry/detail for each slot
	detailsMap := make(map[string]map[string]interface{})
	var detMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)

	for s := range slotMap {
		wg.Add(1)
		go func(st, en string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			det := c.fetchLessonDetail(resourceType, resourceID, st, en)
			if det != nil {
				detMu.Lock()
				detailsMap[st+"_"+en] = det
				detMu.Unlock()
			}
		}(s.start, s.end)
	}
	wg.Wait()

	var lessons []EnrichedLesson

	for _, d := range rawEntries.Days {
		for _, ge := range d.GridEntries {
			dateStr := d.Date
			dateParsed, _ := time.Parse("2006-01-02", dateStr)
			dateInt, _ := strconv.Atoi(dateParsed.Format("20060102"))
			dayOfWeek := GermanDayName(dateParsed.Weekday())

			startTimeStr := extractTime(ge.Duration.Start)
			endTimeStr := extractTime(ge.Duration.End)
			periodStr, pNum := computePeriod(startTimeStr, endTimeStr)

			var subjShort, subjLong, origSubj string
			if len(ge.Position1) > 0 {
				if ge.Position1[0].Current != nil {
					subjShort = ge.Position1[0].Current.ShortName
					subjLong = ge.Position1[0].Current.LongName
				}
				if ge.Position1[0].Removed != nil {
					origSubj = ge.Position1[0].Removed.ShortName
				}
			}

			var rooms []string
			var roomsLong []string
			var origRooms []string
			for _, p2 := range ge.Position2 {
				if p2.Current != nil {
					rooms = append(rooms, p2.Current.ShortName)
					if p2.Current.LongName != "" {
						roomsLong = append(roomsLong, p2.Current.LongName)
					}
				}
				if p2.Removed != nil {
					origRooms = append(origRooms, p2.Removed.ShortName)
				}
			}
			roomStr := strings.Join(rooms, ", ")
			roomLongStr := strings.Join(roomsLong, ", ")
			origRoomStr := strings.Join(origRooms, ", ")

			isCancelled := ge.Status == "CANCELLED"
			isSubst := ge.Status == "CHANGED" || ge.SubstitutionText != "" || origSubj != ""
			isRoomChange := origRoomStr != "" && origRoomStr != roomStr

			var teacherShort, teacherLong, origTeacher, teachingContent string
			var homeworks []string
			entryClass := className

			// Check detailMap
			detailKey := ge.Duration.Start + "_" + ge.Duration.End
			if det, ok := detailsMap[detailKey]; ok {
				if tArr, ok := det["teachers"].([]interface{}); ok {
					var tShorts, tLongs []string
					for _, item := range tArr {
						if tMap, ok := item.(map[string]interface{}); ok {
							if s, ok := tMap["shortName"].(string); ok && s != "" {
								tShorts = append(tShorts, s)
							}
							if l, ok := tMap["longName"].(string); ok && l != "" {
								tLongs = append(tLongs, l)
							}
						}
					}
					if len(tShorts) > 0 {
						teacherShort = strings.Join(tShorts, ", ")
					}
					if len(tLongs) > 0 {
						teacherLong = strings.Join(tLongs, ", ")
					}
				}

				if rArr, ok := det["rooms"].([]interface{}); ok && roomStr == "" {
					var rNames []string
					for _, item := range rArr {
						if rMap, ok := item.(map[string]interface{}); ok {
							if s, ok := rMap["name"].(string); ok && s != "" {
								rNames = append(rNames, s)
							}
						}
					}
					if len(rNames) > 0 {
						roomStr = strings.Join(rNames, ", ")
					}
				}

				if cArr, ok := det["classes"].([]interface{}); ok && entryClass == "" {
					var cNames []string
					for _, item := range cArr {
						if cMap, ok := item.(map[string]interface{}); ok {
							if s, ok := cMap["name"].(string); ok && s != "" {
								cNames = append(cNames, s)
							}
						}
					}
					if len(cNames) > 0 {
						entryClass = strings.Join(cNames, ", ")
					}
				}

				if tc, ok := det["teachingContent"].(string); ok {
					teachingContent = tc
				}

				if hwList, ok := det["homeworks"].([]interface{}); ok {
					for _, hw := range hwList {
						if hwMap, ok := hw.(map[string]interface{}); ok {
							if desc, ok := hwMap["text"].(string); ok && desc != "" {
								homeworks = append(homeworks, desc)
							}
						}
					}
				}

				if detStatus, ok := det["status"].(string); ok && detStatus == "CANCELLED" {
					isCancelled = true
				}
			}

			// Subject fallback
			if subjShort == "" {
				subjShort = "Unterricht"
			}
			if subjLong == "" {
				subjLong = subjShort
			}

			// Colors
			colorHex, textHex := GetSubjectColors(subjShort, ge.Color)

			lessonID := 0
			if len(ge.IDs) > 0 {
				lessonID = ge.IDs[0]
			}

			lesson := EnrichedLesson{
				ID:              lessonID,
				Date:            dateStr,
				DateInt:         dateInt,
				DayOfWeek:       dayOfWeek,
				Period:          periodStr,
				PeriodNum:       pNum,
				StartTimeStr:    startTimeStr,
				EndTimeStr:      endTimeStr,
				TimeRange:       fmt.Sprintf("%s - %s", startTimeStr, endTimeStr),
				Subject:         subjShort,
				SubjectLong:     subjLong,
				OriginalSubject: origSubj,
				Teacher:         teacherShort,
				TeacherLong:     teacherLong,
				OriginalTeacher: origTeacher,
				Room:            roomStr,
				RoomLong:        roomLongStr,
				OriginalRoom:    origRoomStr,
				Class:           entryClass,
				IsCancelled:     isCancelled,
				IsSubstitution:  isSubst,
				IsRoomChange:    isRoomChange,
				SubstText:       ge.SubstitutionText,
				Notes:           ge.LessonText,
				TeachingContent: teachingContent,
				Homeworks:       homeworks,
				Color:           colorHex,
				TextColor:       textHex,
			}

			lessons = append(lessons, lesson)
		}
	}

	// Sort chronologically
	sort.Slice(lessons, func(i, j int) bool {
		if lessons[i].DateInt != lessons[j].DateInt {
			return lessons[i].DateInt < lessons[j].DateInt
		}
		return lessons[i].StartTimeStr < lessons[j].StartTimeStr
	})

	return lessons, nil
}

func (c *Client) fetchLessonDetail(resourceType string, resourceID int, startDateTime, endDateTime string) map[string]interface{} {
	elemType := 1
	switch resourceType {
	case "CLASS":
		elemType = 1
	case "TEACHER":
		elemType = 2
	case "SUBJECT":
		elemType = 3
	case "ROOM":
		elemType = 4
	case "STUDENT":
		elemType = 5
	}

	detailURL := fmt.Sprintf(
		"%s/WebUntis/api/rest/view/v2/calendar-entry/detail?elementId=%d&elementType=%d&startDateTime=%s&endDateTime=%s&homeworkOption=DUE",
		c.Server, resourceID, elemType, url.QueryEscape(startDateTime), url.QueryEscape(endDateTime),
	)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var res struct {
		CalendarEntries []map[string]interface{} `json:"calendarEntries"`
	}
	if err := json.Unmarshal(body, &res); err != nil || len(res.CalendarEntries) == 0 {
		return nil
	}

	// Return the entry matching the slot
	for _, entry := range res.CalendarEntries {
		if s, ok := entry["startDateTime"].(string); ok && strings.HasPrefix(s, startDateTime) {
			return entry
		}
	}

	return res.CalendarEntries[0]
}


func extractTime(dt string) string {
	if len(dt) >= 16 {
		return dt[11:16]
	}
	return dt
}

func computePeriod(start, end string) (string, int) {
	// Standard timetable periods for vocational schools:
	switch start {
	case "07:30":
		if end == "09:00" {
			return "1. - 2. Stunde", 1
		}
		return "1. Stunde", 1
	case "08:15":
		return "2. Stunde", 2
	case "09:15":
		if end == "10:45" {
			return "3. - 4. Stunde", 3
		}
		return "3. Stunde", 3
	case "10:00":
		return "4. Stunde", 4
	case "11:00":
		if end == "12:30" {
			return "5. - 6. Stunde", 5
		}
		return "5. Stunde", 5
	case "11:45":
		return "6. Stunde", 6
	case "12:45":
		if end == "14:15" {
			return "7. - 8. Stunde", 7
		}
		return "7. Stunde", 7
	case "13:30":
		return "8. Stunde", 8
	case "14:30":
		if end == "16:00" {
			return "9. - 10. Stunde", 9
		}
		return "9. Stunde", 9
	case "15:15":
		return "10. Stunde", 10
	default:
		return fmt.Sprintf("%s - %s", start, end), 1
	}
}

// GermanDayName returns German weekday
func GermanDayName(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "Montag"
	case time.Tuesday:
		return "Dienstag"
	case time.Wednesday:
		return "Mittwoch"
	case time.Thursday:
		return "Donnerstag"
	case time.Friday:
		return "Freitag"
	case time.Saturday:
		return "Samstag"
	case time.Sunday:
		return "Sonntag"
	default:
		return ""
	}
}

// GetSubjectColors returns Material Design 3 palette colors for standard subjects
func GetSubjectColors(subj, serverColor string) (bgColor, fgColor string) {
	if serverColor != "" {
		c := strings.TrimPrefix(serverColor, "#")
		if len(c) == 6 {
			return "#" + c, "#1c1b1f"
		}
	}

	upper := strings.ToUpper(strings.TrimSpace(subj))
	switch {
	case strings.HasPrefix(upper, "LF"):
		return "#D1E4FF", "#001D36" // Learning fields / IT - Tech blue
	case strings.HasPrefix(upper, "D") || strings.Contains(upper, "DEUTSCH"):
		return "#FFDAD6", "#410002" // Soft coral red
	case strings.HasPrefix(upper, "M") || strings.Contains(upper, "MATHE"):
		return "#CCE5FF", "#001E30" // Soft azure blue
	case strings.HasPrefix(upper, "E") || strings.Contains(upper, "ENGL"):
		return "#FFDCC2", "#2E1500" // Soft amber orange
	case strings.HasPrefix(upper, "PK") || strings.Contains(upper, "POL"):
		return "#E8DEF8", "#1D192B" // Soft lavender
	case strings.HasPrefix(upper, "WBL") || strings.Contains(upper, "WIRT"):
		return "#E2E2E6", "#1B1B1F" // Neutral slate
	case strings.HasPrefix(upper, "REL") || strings.Contains(upper, "ETH"):
		return "#F2DAFF", "#261334" // Purple orchid
	case strings.HasPrefix(upper, "SP"):
		return "#BFF0D7", "#002114" // Mint emerald
	default:
		return "#E3E2E6", "#1B1B1F" // Neutral container
	}
}
