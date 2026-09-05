package cloud

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benzjeremy/untis-go/db"
)

// Default Client ID for Microsoft Entra ID / Microsoft Identity Platform.
// The default uses the standard multi-tenant public client ID, but users/admins
// can also configure their own custom Azure App Registration Client ID in settings.
const (
	DefaultClientID = "de8bc8b5-d9f9-48b1-a8ad-b8c8da743074"
	DefaultTenant   = "common"
	DefaultScope    = "openid profile email offline_access User.Read Files.ReadWrite"
	OneDrivePath    = "https://graph.microsoft.com/v1.0/me/drive/root:/Apps/untis-go/untis_config.json:/content"
)

// DeviceCodeResponse represents the payload from Microsoft Device Code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// UserInfo represents Microsoft Account user details
type UserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
	AccountType       string `json:"accountType"` // "school_work" or "personal"
}

// MicrosoftClient handles Microsoft OAuth2, Token Management, and OneDrive Sync
type MicrosoftClient struct {
	mu           sync.Mutex
	database     *db.Database
	httpClient   *http.Client
	redirectURI  string
	oauthState   string
	codeVerifier string
}

// NewMicrosoftClient initializes a new MicrosoftClient
func NewMicrosoftClient(database *db.Database, redirectURI string) *MicrosoftClient {
	return &MicrosoftClient{
		database:    database,
		redirectURI: redirectURI,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// SetRedirectURI sets or updates the OAuth redirect URI
func (m *MicrosoftClient) SetRedirectURI(uri string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redirectURI = uri
}

// GetClientID returns the configured Client ID or DefaultClientID
func (m *MicrosoftClient) GetClientID() string {
	if m.database != nil {
		custom := m.database.GetSetting("ms_client_id", "")
		if custom != "" {
			return custom
		}
	}
	return DefaultClientID
}

// GetTenant returns the configured Tenant or "common"
func (m *MicrosoftClient) GetTenant() string {
	if m.database != nil {
		custom := m.database.GetSetting("ms_tenant_id", "")
		if custom != "" {
			return custom
		}
	}
	return DefaultTenant
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generatePKCE() (verifier, challenge string) {
	verifier = generateRandomString(48)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return
}

// GetAuthURL generates an OAuth2 authorization URL with PKCE
func (m *MicrosoftClient) GetAuthURL() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := generateRandomString(24)
	verifier, challenge := generatePKCE()
	m.oauthState = state
	m.codeVerifier = verifier

	tenant := m.GetTenant()
	clientID := m.GetClientID()

	authBase := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant)
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", m.redirectURI)
	params.Set("response_mode", "query")
	params.Set("scope", DefaultScope)
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("prompt", "select_account")

	return fmt.Sprintf("%s?%s", authBase, params.Encode()), nil
}

// HandleCallback exchanges the authorization code for tokens and fetches user info
func (m *MicrosoftClient) HandleCallback(code, state string) (*UserInfo, error) {
	m.mu.Lock()
	savedState := m.oauthState
	verifier := m.codeVerifier
	m.mu.Unlock()

	if savedState == "" || state != savedState {
		return nil, errors.New("ungültiger OAuth2-State (CSRF-Schutz)")
	}

	tenant := m.GetTenant()
	clientID := m.GetClientID()
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", m.redirectURI)
	data.Set("code_verifier", verifier)
	data.Set("scope", DefaultScope)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token-anfrage fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token-austausch fehlgeschlagen (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	// Fetch User Profile from Microsoft Graph
	userInfo, err := m.fetchGraphUserProfile(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("benutzerprofil konnte nicht von Microsoft Graph geladen werden: %w", err)
	}

	// Persist tokens and profile to database
	expiryTime := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
	_ = m.database.SetSetting("ms_access_token", tokenResp.AccessToken)
	_ = m.database.SetSetting("ms_refresh_token", tokenResp.RefreshToken)
	_ = m.database.SetSetting("ms_token_expiry", strconv.FormatInt(expiryTime, 10))
	_ = m.database.SetSetting("ms_user_id", userInfo.ID)
	_ = m.database.SetSetting("ms_user_name", userInfo.DisplayName)
	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}
	_ = m.database.SetSetting("ms_user_email", email)
	_ = m.database.SetSetting("ms_account_type", userInfo.AccountType)
	_ = m.database.SetSetting("ms_logged_in", "true")

	return userInfo, nil
}

// StartDeviceCode begins the Microsoft Device Code Flow
func (m *MicrosoftClient) StartDeviceCode() (*DeviceCodeResponse, error) {
	tenant := m.GetTenant()
	clientID := m.GetClientID()
	deviceCodeURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", tenant)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", DefaultScope)

	req, err := http.NewRequest("POST", deviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device-code anfrage fehlgeschlagen (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var dResp DeviceCodeResponse
	if err := json.Unmarshal(body, &dResp); err != nil {
		return nil, err
	}

	return &dResp, nil
}

// PollDeviceCode polls the token endpoint for device code completion
func (m *MicrosoftClient) PollDeviceCode(deviceCode string) (*UserInfo, error) {
	tenant := m.GetTenant()
	clientID := m.GetClientID()
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", deviceCode)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &errResp)
		if errResp.Error == "authorization_pending" {
			return nil, errors.New("authorization_pending")
		} else if errResp.Error == "slow_down" {
			return nil, errors.New("slow_down")
		} else if errResp.Error == "code_expired" {
			return nil, errors.New("code_expired")
		}
		return nil, fmt.Errorf("device-code login fehler: %s (%s)", errResp.Error, errResp.ErrorDescription)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	userInfo, err := m.fetchGraphUserProfile(tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}

	expiryTime := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
	_ = m.database.SetSetting("ms_access_token", tokenResp.AccessToken)
	_ = m.database.SetSetting("ms_refresh_token", tokenResp.RefreshToken)
	_ = m.database.SetSetting("ms_token_expiry", strconv.FormatInt(expiryTime, 10))
	_ = m.database.SetSetting("ms_user_id", userInfo.ID)
	_ = m.database.SetSetting("ms_user_name", userInfo.DisplayName)
	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}
	_ = m.database.SetSetting("ms_user_email", email)
	_ = m.database.SetSetting("ms_account_type", userInfo.AccountType)
	_ = m.database.SetSetting("ms_logged_in", "true")

	return userInfo, nil
}

func (m *MicrosoftClient) fetchGraphUserProfile(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("graph api status %d: %s", resp.StatusCode, string(body))
	}

	var u UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}

	// Detect account type: School/Work (Entra ID) vs Personal MSA
	// In Microsoft Graph, personal accounts typically have @outlook, @live, @hotmail or specific UPN formats.
	// School/Organization accounts usually have custom domain or onmicrosoft.com
	upn := strings.ToLower(u.UserPrincipalName)
	if strings.Contains(upn, "live.com") || strings.Contains(upn, "outlook.com") || strings.Contains(upn, "hotmail.com") {
		u.AccountType = "personal"
	} else {
		u.AccountType = "school_work"
	}

	return &u, nil
}

// GetValidAccessToken returns the current access token, renewing it via refresh_token if expired
func (m *MicrosoftClient) GetValidAccessToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accessToken := m.database.GetSetting("ms_access_token", "")
	refreshToken := m.database.GetSetting("ms_refresh_token", "")
	expiryStr := m.database.GetSetting("ms_token_expiry", "0")

	if accessToken == "" || refreshToken == "" {
		return "", errors.New("kein Microsoft-Konto angemeldet")
	}

	expiryUnix, _ := strconv.ParseInt(expiryStr, 10, 64)
	// If token expires in less than 2 minutes, refresh it proactively
	if time.Now().Unix()+120 >= expiryUnix {
		tenant := m.GetTenant()
		clientID := m.GetClientID()
		tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

		data := url.Values{}
		data.Set("client_id", clientID)
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", refreshToken)
		data.Set("scope", DefaultScope)

		req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return accessToken, nil
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return accessToken, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var tokenResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				ExpiresIn    int    `json:"expires_in"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err == nil {
				newExpiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
				accessToken = tokenResp.AccessToken
				_ = m.database.SetSetting("ms_access_token", tokenResp.AccessToken)
				if tokenResp.RefreshToken != "" {
					_ = m.database.SetSetting("ms_refresh_token", tokenResp.RefreshToken)
				}
				_ = m.database.SetSetting("ms_token_expiry", strconv.FormatInt(newExpiry, 10))
			}
		}
	}

	return accessToken, nil
}

// IsAuthenticated checks if a user is actively authenticated with Microsoft
func (m *MicrosoftClient) IsAuthenticated() bool {
	loggedIn := m.database.GetSetting("ms_logged_in", "false") == "true"
	token := m.database.GetSetting("ms_access_token", "")
	return loggedIn && token != ""
}

// Logout clears stored Microsoft authentication tokens and credentials
func (m *MicrosoftClient) Logout() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = m.database.SetSetting("ms_logged_in", "false")
	_ = m.database.SetSetting("ms_access_token", "")
	_ = m.database.SetSetting("ms_refresh_token", "")
	_ = m.database.SetSetting("ms_token_expiry", "0")
	_ = m.database.SetSetting("ms_user_id", "")
	_ = m.database.SetSetting("ms_user_name", "")
	_ = m.database.SetSetting("ms_user_email", "")
	_ = m.database.SetSetting("ms_account_type", "")
	return nil
}

// UploadConfigToOneDrive uploads user's configuration JSON to their OneDrive
func (m *MicrosoftClient) UploadConfigToOneDrive(data []byte) error {
	accessToken, err := m.GetValidAccessToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", OneDrivePath, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("onedrive upload fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onedrive upload fehlgeschlagen (HTTP %d): %s", resp.StatusCode, string(body))
	}

	nowISO := time.Now().Format(time.RFC3339)
	_ = m.database.SetSetting("ms_last_sync", nowISO)

	return nil
}

// DownloadConfigFromOneDrive downloads user's configuration JSON from their OneDrive
func (m *MicrosoftClient) DownloadConfigFromOneDrive() ([]byte, error) {
	accessToken, err := m.GetValidAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", OneDrivePath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("onedrive download fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("keine gesicherten Konfigurationen in OneDrive gefunden")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("onedrive download fehlgeschlagen (HTTP %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	nowISO := time.Now().Format(time.RFC3339)
	_ = m.database.SetSetting("ms_last_sync", nowISO)

	return body, nil
}
