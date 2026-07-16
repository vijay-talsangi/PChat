package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type UserInfo struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	PublicKey        string `json:"public_key"`
	SigningPublicKey string `json:"signing_public_key"`
	CreatedAt        string `json:"created_at"`
}

type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type RoomData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
}

type CreateRoomResponse struct {
	Room RoomData `json:"room"`
}

type RoomListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
	MemberCount int    `json:"member_count"`
}

type ListRoomsResponse struct {
	Rooms []RoomListItem `json:"rooms"`
}

type JoinResponse struct {
	Message string `json:"message"`
	RoomID  string `json:"room_id"`
}

type InviteData struct {
	ID        string `json:"id"`
	RoomID    string `json:"room_id"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
	MaxUses   int    `json:"max_uses"`
	Uses      int    `json:"uses"`
}

type CreateInviteResponse struct {
	Invite InviteData `json:"invite"`
}

type MemberInfo struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	PublicKey        string `json:"public_key"`
	SigningPublicKey string `json:"signing_public_key"`
}

type MembersResponse struct {
	Members []MemberInfo `json:"members"`
}

type TurnCreds struct {
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
	TTL        int      `json:"ttl"`
	URLs       []string `json:"urls"`
}

type TurnResponse struct {
	Credentials TurnCreds `json:"credentials"`
}

type RoomKeyResponse struct {
	EncryptedRoomKey string `json:"encrypted_room_key"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type apiError struct {
	Error string `json:"error"`
}

type HTTPStatusError struct {
	StatusCode int
	Message    string
}

func (e *HTTPStatusError) Error() string {
	return e.Message
}

type apiEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type Client struct {
	BaseURL    string
	Token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Register(username, password, publicKey, signingPublicKey string) (*AuthResponse, error) {
	body := map[string]string{
		"username":           username,
		"password":           password,
		"public_key":         publicKey,
		"signing_public_key": signingPublicKey,
	}
	var resp AuthResponse
	if err := c.post("/api/register", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Login(username, password string) (*AuthResponse, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	var resp AuthResponse
	if err := c.post("/api/login", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateRoom(name, encryptedRoomKey string) (*RoomData, error) {
	body := map[string]string{
		"name":               name,
		"encrypted_room_key": encryptedRoomKey,
	}
	var resp CreateRoomResponse
	if err := c.post("/api/rooms", body, &resp); err != nil {
		return nil, err
	}
	return &resp.Room, nil
}

func (c *Client) ListRooms() ([]RoomListItem, error) {
	var resp ListRoomsResponse
	if err := c.get("/api/rooms", &resp); err != nil {
		return nil, err
	}
	return resp.Rooms, nil
}

func (c *Client) JoinRoom(code string) (*JoinResponse, error) {
	body := map[string]string{
		"code": code,
	}
	var resp JoinResponse
	if err := c.post("/api/rooms/join", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) LeaveRoom(name string) error {
	return c.postNoBody("/api/rooms/" + url.PathEscape(name) + "/leave")
}

func (c *Client) DeleteRoom(name string) error {
	return c.delete("/api/rooms/" + url.PathEscape(name))
}

func (c *Client) CreateInvite(roomName string, maxUses, expiresInHours int) (*InviteData, error) {
	body := map[string]int{
		"max_uses":         maxUses,
		"expires_in_hours": expiresInHours,
	}
	var resp CreateInviteResponse
	if err := c.post("/api/rooms/"+url.PathEscape(roomName)+"/invite", body, &resp); err != nil {
		return nil, err
	}
	return &resp.Invite, nil
}

func (c *Client) GetRoomMembers(roomName string) ([]MemberInfo, error) {
	var resp MembersResponse
	if err := c.get("/api/rooms/"+url.PathEscape(roomName)+"/members", &resp); err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (c *Client) GetTurnCredentials(roomName string) (*TurnCreds, error) {
	var resp TurnResponse
	if err := c.get("/api/rooms/"+url.PathEscape(roomName)+"/turn-credentials", &resp); err != nil {
		return nil, err
	}
	return &resp.Credentials, nil
}

func (c *Client) GetRoomKey(roomName string) (string, error) {
	var resp RoomKeyResponse
	if err := c.get("/api/rooms/"+url.PathEscape(roomName)+"/key", &resp); err != nil {
		return "", err
	}
	return resp.EncryptedRoomKey, nil
}

func (c *Client) UploadRoomKey(roomName, userID, encryptedRoomKey string) error {
	body := map[string]string{
		"user_id":            userID,
		"encrypted_room_key": encryptedRoomKey,
	}
	return c.post("/api/rooms/"+url.PathEscape(roomName)+"/keys", body, nil)
}

func (c *Client) GetMembersWithoutKeys(roomName string) ([]MemberInfo, error) {
	var resp MembersResponse
	if err := c.get("/api/rooms/"+url.PathEscape(roomName)+"/keys/missing", &resp); err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (c *Client) get(path string, result interface{}) error {
	reqURL := c.BaseURL + path
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	reqURL := c.BaseURL + path
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

func (c *Client) postNoBody(path string) error {
	reqURL := c.BaseURL + path
	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	return nil
}

func (c *Client) delete(path string) error {
	reqURL := c.BaseURL + path
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("failed to decode response envelope: %w (body: %s)", err, string(body))
	}
	if result != nil {
		if err := json.Unmarshal(env.Data, result); err != nil {
			return fmt.Errorf("failed to decode response data: %w (data: %s)", err, string(env.Data))
		}
	}
	return nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("server returned status %d", resp.StatusCode)}
	}
	var apiErr apiError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("server error (%d): %s", resp.StatusCode, apiErr.Error)}
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("server error (%d): %s", resp.StatusCode, string(body))}
}
