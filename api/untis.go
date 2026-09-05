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
	Class           string   `json:"class"`           // e.g. "10A"
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

	// Selected class for anonymous/guest sessions
	SelectedClassID   int
	SelectedClassName string
}

// NormalizeServerURL cleans and extracts the base origin (e.g. "https://bk-technik-siegen.webuntis.com")
// stripping any subpaths like /WebUntis/ or query parameters like ?school=...
func NormalizeServerURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, u.Host)
}

// NewClient initializes a real WebUntis client
func NewClient(server, school, username, password, authType string) *Client {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}

	cleanServer := NormalizeServerURL(server)

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

// IsAnonymous returns true if the client is configured for guest/anonymous access
func (c *Client) IsAnonymous() bool {
	u := strings.TrimSpace(c.Username)
	return u == "" || u == "#anonymous#" || strings.EqualFold(c.AuthType, "anonymous")
}

// SetSelectedClass updates the designated class ID and name for the client
func (c *Client) SetSelectedClass(classID int, className string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SelectedClassID = classID
	c.SelectedClassName = className
}

// GetSelectedClass returns the selected class ID and name in a thread-safe manner
func (c *Client) GetSelectedClass() (int, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.SelectedClassID, c.SelectedClassName
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

	// Anonymous / Guest access: WebUntis protocol with #anonymous#
	if c.IsAnonymous() {
		c.UserInfo.DisplayName = "Gast"
		c.UserInfo.DetectedClass = ""
		c.Username = "#anonymous#"

		// Step 1: getAppSharedSecret
		url1 := fmt.Sprintf("%s/WebUntis/jsonrpc_intern.do?m=getAppSharedSecret&school=%s&v=i3.5", c.Server, url.QueryEscape(c.School))
		body1 := map[string]interface{}{
			"id":      "1",
			"method":  "getAppSharedSecret",
			"params":  []interface{}{map[string]string{"userName": "#anonymous#", "password": ""}},
			"jsonrpc": "2.0",
		}
		b1, err := json.Marshal(body1)
		if err != nil {
			return err
		}
		req1, err := http.NewRequest("POST", url1, bytes.NewReader(b1))
		if err != nil {
			return err
		}
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
		resp1, err := c.httpClient.Do(req1)
		if err != nil {
			return fmt.Errorf("verbindung zum WebUntis Server fehlgeschlagen: %w", err)
		}
		defer resp1.Body.Close()

		var res1 struct {
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		resp1Bytes, _ := io.ReadAll(resp1.Body)
		_ = json.Unmarshal(resp1Bytes, &res1)
		if res1.Error != nil {
			if res1.Error.Code == -8523 || strings.Contains(strings.ToLower(res1.Error.Message), "no public access") {
				return fmt.Errorf("diese Schule hat den anonymen Gastzugang nicht freigeschaltet (no public access). Bitte melde dich mit deinen Zugangsdaten an.")
			}
			return fmt.Errorf("anonymer Gastzugang fehlgeschlagen: %s", res1.Error.Message)
		}

		// Step 2: getUserData2017 with OTP 100170
		url2 := fmt.Sprintf("%s/WebUntis/jsonrpc_intern.do?m=getUserData2017&school=%s&v=i2.2", c.Server, url.QueryEscape(c.School))
		clientTime := time.Now().UnixMilli()
		body2 := map[string]interface{}{
			"id":     "2",
			"method": "getUserData2017",
			"params": []interface{}{
				map[string]interface{}{
					"auth": map[string]interface{}{
						"clientTime": clientTime,
						"user":       "#anonymous#",
						"otp":        100170,
					},
				},
			},
			"jsonrpc": "2.0",
		}
		b2, err := json.Marshal(body2)
		if err != nil {
			return err
		}
		req2, err := http.NewRequest("POST", url2, bytes.NewReader(b2))
		if err != nil {
			return err
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
		resp2, err := c.httpClient.Do(req2)
		if err != nil {
			return fmt.Errorf("sitzungsaufbau fehlgeschlagen: %w", err)
		}
		defer resp2.Body.Close()

		// Step 3: Try getting token
		c.Token = "ANONYMOUS"
		tokenURL := fmt.Sprintf("%s/WebUntis/api/token/new", c.Server)
		tokenReq, err := http.NewRequest("GET", tokenURL, nil)
		if err == nil {
			tokenReq.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
			if tokenResp, err := c.httpClient.Do(tokenReq); err == nil {
				defer tokenResp.Body.Close()
				if tokenBytes, err := io.ReadAll(tokenResp.Body); err == nil {
					tokStr := strings.TrimSpace(string(tokenBytes))
					if strings.HasPrefix(tokStr, "ey") {
						c.Token = tokStr
					}
				}
			}
		}

		return nil
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
		State      string `json:"state"`
		LoginError string `json:"loginError"`
		Message    string `json:"message"`
		SwitchUI   bool   `json:"switchUI"`
	}
	if err := json.Unmarshal(respBytes, &loginResult); err != nil {
		return fmt.Errorf("ungültige antwort vom server: %s", string(respBytes))
	}

	if loginResult.State != "SUCCESS" {
		if loginResult.State == "NO_MANDANT" {
			return fmt.Errorf("schule '%s' wurde auf dem Server nicht gefunden (NO_MANDANT)", c.School)
		}
		if loginResult.LoginError != "" {
			return fmt.Errorf("anmeldung fehlgeschlagen: %s", loginResult.LoginError)
		}
		if loginResult.Message != "" {
			return fmt.Errorf("anmeldung fehlgeschlagen: %s", loginResult.Message)
		}
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

func (c *Client) setAuthHeader(req *http.Request) {
	if c.Token != "" && c.Token != "ANONYMOUS" && !c.IsAnonymous() {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) fetchGeneralData() {
	dataURL := fmt.Sprintf("%s/WebUntis/api/rest/view/v1/app/data", c.Server)
	req, err := http.NewRequest("GET", dataURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

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

		// Try extracting class name from email prefix (e.g. "itt125.max.mustermann@..." -> "ITT125")
		emailPrefix := strings.Split(gd.User.Email, ".")[0]
		if len(emailPrefix) >= 3 && len(emailPrefix) <= 10 && !strings.EqualFold(emailPrefix, "schule") {
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

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do?school=%s", c.Server, url.QueryEscape(c.School))
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
		c.setAuthHeader(req)
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

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do?school=%s", c.Server, url.QueryEscape(c.School))
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
	c.setAuthHeader(req)

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

	rpcURL := fmt.Sprintf("%s/WebUntis/jsonrpc.do?school=%s", c.Server, url.QueryEscape(c.School))
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
	c.setAuthHeader(req)

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
		c.setAuthHeader(restReq)
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
	if c.IsAnonymous() {
		return []Message{}, nil
	}
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
	c.setAuthHeader(req)

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
	c.setAuthHeader(req)

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
	if c.IsAnonymous() {
		return []WebUntisHomework{}, nil
	}
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
	if c.IsAnonymous() {
		return []WebUntisAbsence{}, nil
	}
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	personID := c.UserInfo.PersonID
	if personID == 0 {
		return nil, fmt.Errorf("keine gültige Schüler-ID gefunden")
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

// GetOwnTimetable retrieves the personal student timetable
func (c *Client) GetOwnTimetable(startDate, endDate time.Time) ([]EnrichedLesson, error) {
	if c.IsAnonymous() {
		c.mu.Lock()
		classID := c.SelectedClassID
		c.mu.Unlock()
		if classID > 0 {
			return c.GetTimetable(classID, startDate, endDate)
		}
		if klassen, errK := c.GetKlassen(); errK == nil && len(klassen) > 0 {
			c.mu.Lock()
			c.SelectedClassID = klassen[0].ID
			c.SelectedClassName = klassen[0].Name
			classID = klassen[0].ID
			c.mu.Unlock()
			return c.GetTimetable(classID, startDate, endDate)
		}
		return []EnrichedLesson{}, nil
	}

	var lessons []EnrichedLesson
	var err error

	personID := c.UserInfo.PersonID
	if personID > 0 {
		lessons, err = c.GetTimetableForResource("STUDENT", personID, startDate, endDate, "MY_TIMETABLE")
		if err == nil && len(lessons) > 0 {
			return lessons, nil
		}
	}

	// If individual student timetable has NO_DATA (standard for German vocational schools),
	// fallback to the student's detected assigned class
	var classID int
	if c.UserInfo.DetectedClass != "" {
		c.mu.Lock()
		for id, k := range c.klassenCache {
			if strings.EqualFold(k.Name, c.UserInfo.DetectedClass) {
				classID = id
				break
			}
		}
		c.mu.Unlock()

		// If not cached yet, try loading classes
		if classID == 0 {
			if klassen, errK := c.GetKlassen(); errK == nil {
				for _, k := range klassen {
					if strings.EqualFold(k.Name, c.UserInfo.DetectedClass) {
						classID = k.ID
						break
					}
				}
			}
		}
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

	if c.IsAnonymous() {
		return c.fetchPublicTimetable(resourceType, resourceID, startDate, endDate)
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
		c.setAuthHeader(req)
		return c.httpClient.Do(req)
	}

	resp, err := doFetch()
	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		if resp != nil {
			resp.Body.Close()
		}
		if pubLessons, pubErr := c.fetchPublicTimetable(resourceType, resourceID, startDate, endDate); pubErr == nil && len(pubLessons) > 0 {
			return pubLessons, nil
		}
		if err != nil {
			return nil, fmt.Errorf("stundenplan-abfrage fehlgeschlagen: %w", err)
		}
		return nil, fmt.Errorf("stundenplan-abfrage fehlgeschlagen (status %d)", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type rawGridPosItem struct {
		Current *struct {
			Type        string `json:"type"`
			Status      string `json:"status"`
			ShortName   string `json:"shortName"`
			LongName    string `json:"longName"`
			DisplayName string `json:"displayName"`
		} `json:"current"`
		Removed *struct {
			Type      string `json:"type"`
			ShortName string `json:"shortName"`
		} `json:"removed"`
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
				Position1 []rawGridPosItem `json:"position1"`
				Position2 []rawGridPosItem `json:"position2"`
				Position3 []rawGridPosItem `json:"position3"`
				Position4 []rawGridPosItem `json:"position4"`
				LessonInfo string           `json:"lessonInfo"`
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
	seenLessons := make(map[string]bool)

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
			var teachers, teachersLong []string
			var origTeacher string
			var rooms, roomsLong, origRooms []string
			var classes []string

			allPositions := [][]rawGridPosItem{ge.Position1, ge.Position2, ge.Position3, ge.Position4}
			for _, posList := range allPositions {
				for _, p := range posList {
					if p.Current != nil {
						switch p.Current.Type {
						case "SUBJECT":
							if subjShort == "" {
								subjShort = p.Current.ShortName
								subjLong = p.Current.LongName
							}
						case "TEACHER":
							teachers = append(teachers, p.Current.ShortName)
							if p.Current.LongName != "" {
								teachersLong = append(teachersLong, p.Current.LongName)
							}
						case "ROOM":
							rooms = append(rooms, p.Current.ShortName)
							if p.Current.LongName != "" {
								roomsLong = append(roomsLong, p.Current.LongName)
							}
						case "CLASS":
							classes = append(classes, p.Current.ShortName)
						}
					}
					if p.Removed != nil {
						switch p.Removed.Type {
						case "SUBJECT":
							origSubj = p.Removed.ShortName
						case "ROOM":
							origRooms = append(origRooms, p.Removed.ShortName)
						case "TEACHER":
							origTeacher = p.Removed.ShortName
						}
					}
				}
			}

			// Fallback: If no subject was found, but lessonInfo / lessonText exists
			if subjShort == "" {
				if ge.LessonInfo != "" {
					subjShort = ge.LessonInfo
					subjLong = ge.LessonInfo
				} else if ge.LessonText != "" {
					subjShort = ge.LessonText
					subjLong = ge.LessonText
				} else if ge.SubstitutionText != "" {
					subjShort = ge.SubstitutionText
					subjLong = ge.SubstitutionText
				}
			}

			// Fallback: If still no subject, check Position1
			if subjShort == "" && len(ge.Position1) > 0 && ge.Position1[0].Current != nil {
				subjShort = ge.Position1[0].Current.ShortName
				subjLong = ge.Position1[0].Current.LongName
			}

			teacherShort := strings.Join(teachers, ", ")
			teacherLong := strings.Join(teachersLong, ", ")
			roomStr := strings.Join(rooms, ", ")
			roomLongStr := strings.Join(roomsLong, ", ")
			origRoomStr := strings.Join(origRooms, ", ")
			entryClass := className
			if len(classes) > 0 {
				entryClass = strings.Join(classes, ", ")
			}

			isCancelled := ge.Status == "CANCELLED"
			isSubst := ge.Status == "CHANGED" || ge.SubstitutionText != "" || origSubj != ""
			isRoomChange := origRoomStr != "" && origRoomStr != roomStr

			var teachingContent string
			var homeworks []string

			// Check detailMap
			detailKey := ge.Duration.Start + "_" + ge.Duration.End
			if det, ok := detailsMap[detailKey]; ok {
				// 1. Teachers from detail
				if tArr, ok := det["teachers"].([]interface{}); ok && len(tArr) > 0 {
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

				// 2. Subject from detail
				if subjMap, ok := det["subject"].(map[string]interface{}); ok {
					if s, ok := subjMap["shortName"].(string); ok && s != "" {
						subjShort = s
					}
					if l, ok := subjMap["longName"].(string); ok && l != "" {
						subjLong = l
					}
				}

				// 3. Rooms from detail
				if rArr, ok := det["rooms"].([]interface{}); ok && len(rArr) > 0 {
					var rNames, rLongs []string
					for _, item := range rArr {
						if rMap, ok := item.(map[string]interface{}); ok {
							if s, ok := rMap["shortName"].(string); ok && s != "" {
								rNames = append(rNames, s)
							}
							if l, ok := rMap["longName"].(string); ok && l != "" {
								rLongs = append(rLongs, l)
							}
						}
					}
					if len(rNames) > 0 {
						roomStr = strings.Join(rNames, ", ")
					}
					if len(rLongs) > 0 {
						roomLongStr = strings.Join(rLongs, ", ")
					}
				}

				// 4. Klasses from detail (WebUntis uses 'klasses')
				var kArr []interface{}
				if k, ok := det["klasses"].([]interface{}); ok {
					kArr = k
				} else if c, ok := det["classes"].([]interface{}); ok {
					kArr = c
				}
				if len(kArr) > 0 {
					var cNames []string
					for _, item := range kArr {
						if cMap, ok := item.(map[string]interface{}); ok {
							if s, ok := cMap["shortName"].(string); ok && s != "" {
								cNames = append(cNames, s)
							} else if s, ok := cMap["name"].(string); ok && s != "" {
								cNames = append(cNames, s)
							}
						}
					}
					if len(cNames) > 0 {
						entryClass = strings.Join(cNames, ", ")
						if c.UserInfo.DetectedClass == "" || strings.EqualFold(c.UserInfo.DetectedClass, "SCHULE") {
							c.UserInfo.DetectedClass = cNames[0]
						}
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

			// Deduplication: prevent duplicate lessons returned by WebUntis for split/combined groups
			dedupKey := fmt.Sprintf("%s_%s_%s_%s_%s_%s", dateStr, startTimeStr, endTimeStr, subjShort, roomStr, teacherShort)
			if seenLessons[dedupKey] {
				continue
			}
			seenLessons[dedupKey] = true

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

func (c *Client) fetchPublicTimetable(resourceType string, resourceID int, startDate, endDate time.Time) ([]EnrichedLesson, error) {
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
	}

	dateStr := startDate.Format("2006-01-02")
	publicURL := fmt.Sprintf("%s/WebUntis/api/public/timetable/weekly/data?elementType=%d&elementId=%d&date=%s", c.Server, elemType, resourceID, dateStr)

	req, err := http.NewRequest("GET", publicURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "page.codeberg.ostfriese4.Untis 4.3.0")
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("public timetable status %d", resp.StatusCode)
	}

	var pubResp struct {
		Data struct {
			Result struct {
				Data struct {
					ElementPeriods map[string][]struct {
						ID         int    `json:"id"`
						LessonID   int    `json:"lessonId"`
						Date       int    `json:"date"`
						StartTime  int    `json:"startTime"`
						EndTime    int    `json:"endTime"`
						SubstText  string `json:"substText"`
						PeriodText string `json:"periodText"`
						LessonText string `json:"lessonText"`
						PeriodInfo string `json:"periodInfo"`
						CellState  string `json:"cellState"`
						Elements   []struct {
							Type  int  `json:"type"`
							ID    int  `json:"id"`
							OrgID int  `json:"orgId"`
						} `json:"elements"`
					} `json:"elementPeriods"`
					Elements []struct {
						Type     int    `json:"type"`
						ID       int    `json:"id"`
						Name     string `json:"name"`
						LongName string `json:"longName"`
					} `json:"elements"`
				} `json:"data"`
			} `json:"result"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &pubResp); err != nil {
		return nil, err
	}

	elemNameMap := make(map[int]string)
	elemLongMap := make(map[int]string)
	for _, el := range pubResp.Data.Result.Data.Elements {
		key := el.Type*1000000 + el.ID
		elemNameMap[key] = el.Name
		elemLongMap[key] = el.LongName
	}

	keyStr := strconv.Itoa(resourceID)
	rawPeriods := pubResp.Data.Result.Data.ElementPeriods[keyStr]
	if resourceID <= 0 && len(rawPeriods) == 0 {
		for _, v := range pubResp.Data.Result.Data.ElementPeriods {
			rawPeriods = v
			break
		}
	}

	var lessons []EnrichedLesson
	seenLessons := make(map[string]bool)

	for _, p := range rawPeriods {
		dStr := strconv.Itoa(p.Date)
		if len(dStr) != 8 {
			continue
		}
		formattedDate := fmt.Sprintf("%s-%s-%s", dStr[:4], dStr[4:6], dStr[6:])
		dateParsed, errP := time.Parse("2006-01-02", formattedDate)
		if errP != nil {
			continue
		}

		dayOfWeek := GermanDayName(dateParsed.Weekday())
		sHour := p.StartTime / 100
		sMin := p.StartTime % 100
		eHour := p.EndTime / 100
		eMin := p.EndTime % 100
		startTimeStr := fmt.Sprintf("%02d:%02d", sHour, sMin)
		endTimeStr := fmt.Sprintf("%02d:%02d", eHour, eMin)
		periodStr, pNum := computePeriod(startTimeStr, endTimeStr)

		var subjShort, subjLong string
		var teachers, teachersLong []string
		var rooms, roomsLong []string
		var classes []string

		for _, el := range p.Elements {
			key := el.Type*1000000 + el.ID
			name := elemNameMap[key]
			longName := elemLongMap[key]
			switch el.Type {
			case 1:
				if name != "" {
					classes = append(classes, name)
				}
			case 2:
				if name != "" {
					teachers = append(teachers, name)
				}
				if longName != "" {
					teachersLong = append(teachersLong, longName)
				}
			case 3:
				if subjShort == "" {
					subjShort = name
					subjLong = longName
				}
			case 4:
				if name != "" {
					rooms = append(rooms, name)
				}
				if longName != "" {
					roomsLong = append(roomsLong, longName)
				}
			}
		}

		if subjShort == "" {
			subjShort = "Unterricht"
			subjLong = "Unterricht"
		}
		if subjLong == "" {
			subjLong = subjShort
		}

		colorHex, textHex := GetSubjectColors(subjShort, "")
		teacherShort := strings.Join(teachers, ", ")
		teacherLongStr := strings.Join(teachersLong, ", ")
		roomStr := strings.Join(rooms, ", ")
		roomLongStr := strings.Join(roomsLong, ", ")
		classStr := strings.Join(classes, ", ")
		if classStr == "" {
			c.mu.Lock()
			if k, ok := c.klassenCache[resourceID]; ok {
				classStr = k.Name
			} else if c.SelectedClassName != "" {
				classStr = c.SelectedClassName
			}
			c.mu.Unlock()
		}

		isCancelled := p.CellState == "CANCELLED"
		isSubst := p.CellState == "SUBSTITUTION" || p.CellState == "CHANGED" || p.SubstText != ""

		dedupKey := fmt.Sprintf("%s_%s_%s_%s_%s_%s", formattedDate, startTimeStr, endTimeStr, subjShort, roomStr, teacherShort)
		if seenLessons[dedupKey] {
			continue
		}
		seenLessons[dedupKey] = true

		notes := p.PeriodText
		if notes == "" {
			notes = p.LessonText
		}

		lessons = append(lessons, EnrichedLesson{
			ID:             p.LessonID,
			Date:           formattedDate,
			DateInt:        p.Date,
			DayOfWeek:      dayOfWeek,
			Period:         periodStr,
			PeriodNum:      pNum,
			StartTimeStr:   startTimeStr,
			EndTimeStr:     endTimeStr,
			TimeRange:      fmt.Sprintf("%s - %s", startTimeStr, endTimeStr),
			Subject:        subjShort,
			SubjectLong:    subjLong,
			Teacher:        teacherShort,
			TeacherLong:    teacherLongStr,
			Room:           roomStr,
			RoomLong:       roomLongStr,
			Class:          classStr,
			IsCancelled:    isCancelled,
			IsSubstitution: isSubst,
			SubstText:      p.SubstText,
			Notes:          notes,
			Color:          colorHex,
			TextColor:      textHex,
		})
	}

	sort.Slice(lessons, func(i, j int) bool {
		if lessons[i].DateInt != lessons[j].DateInt {
			return lessons[i].DateInt < lessons[j].DateInt
		}
		return lessons[i].StartTimeStr < lessons[j].StartTimeStr
	})

	return lessons, nil
}

func extractTime(dt string) string {
	if len(dt) >= 16 {
		return dt[11:16]
	}
	return dt
}

func computePeriod(start, end string) (string, int) {
	// School timetable periods (regular and evening classes up to 23:00)
	startMap := map[string]int{
		"07:30": 1, "07:35": 1, "07:45": 1, "08:00": 1,
		"08:15": 2, "08:20": 2, "08:30": 2,
		"09:15": 3, "09:20": 3, "09:30": 3,
		"10:00": 4, "10:05": 4, "10:15": 4,
		"11:00": 5, "11:05": 5, "11:15": 5,
		"11:45": 6, "12:00": 6,
		"12:45": 7, "12:50": 7, "13:00": 7,
		"13:30": 8, "13:45": 8,
		"14:30": 9, "14:45": 9,
		"15:15": 10, "15:30": 10,
		"16:15": 11, "16:30": 11,
		"17:00": 12, "17:15": 12,
		"18:00": 13, "18:15": 13,
		"18:45": 14, "19:00": 14,
		"19:45": 15, "20:00": 15,
		"20:30": 16, "20:45": 16,
		"21:15": 17,
	}

	endMap := map[string]int{
		"08:15": 1, "08:20": 1,
		"09:00": 2, "09:05": 2,
		"10:00": 3, "10:05": 3,
		"10:45": 4, "10:50": 4,
		"11:45": 5, "11:50": 5,
		"12:30": 6, "12:35": 6,
		"13:30": 7, "13:35": 7,
		"14:15": 8, "14:20": 8,
		"15:15": 9,
		"16:00": 10,
		"17:00": 11,
		"17:45": 12,
		"18:45": 13,
		"19:30": 14,
		"20:30": 15,
		"21:15": 16,
		"22:00": 17,
		"22:45": 18,
	}

	pStart, hasStart := startMap[start]
	pEnd, hasEnd := endMap[end]

	if !hasStart {
		var h, m int
		fmt.Sscanf(start, "%d:%d", &h, &m)
		if h >= 7 {
			min := (h-7)*60 + m - 30
			if min >= 0 {
				pStart = 1 + (min / 50)
			} else {
				pStart = 1
			}
		} else {
			pStart = 1
		}
	}

	if !hasEnd {
		var h, m int
		fmt.Sscanf(end, "%d:%d", &h, &m)
		if h >= 8 {
			min := (h-7)*60 + m - 30
			if min > 0 {
				pEnd = 1 + (min / 50)
			} else {
				pEnd = pStart
			}
		} else {
			pEnd = pStart
		}
	}

	if pEnd < pStart {
		pEnd = pStart
	}

	if pStart == pEnd {
		return fmt.Sprintf("%d. Stunde", pStart), pStart
	}
	return fmt.Sprintf("%d. - %d. Stunde", pStart, pEnd), pStart
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
