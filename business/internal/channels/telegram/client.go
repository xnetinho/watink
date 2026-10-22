package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type GetMeResponse struct {
	Ok     bool `json:"ok"`
	Result User `json:"result"`
}

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("telegram api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res GetMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("telegram getMe not ok")
	}
	return &res.Result, nil
}

type SetWebhookRequest struct {
	URL            string   `json:"url"`
	SecretToken    string   `json:"secret_token,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL, secretToken string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", c.token)
	payload := SetWebhookRequest{
		URL:            webhookURL,
		SecretToken:    secretToken,
		AllowedUpdates: []string{"message", "edited_message", "my_chat_member"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("setWebhook error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type GetUpdatesResponse struct {
	Ok     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout int) ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d", c.token, offset, timeout)
	httpClient := &http.Client{
		Timeout: time.Duration(timeout+5) * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("getUpdates error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res GetUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("telegram getUpdates not ok")
	}
	return res.Result, nil
}

type SendMessageRequest struct {
	ChatID           interface{} `json:"chat_id"` // int64 or string
	Text             string      `json:"text"`
	ReplyToMessageID int         `json:"reply_to_message_id,omitempty"`
}

type TelegramResponse struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description,omitempty"`
}

type MessageResult struct {
	MessageID int64 `json:"message_id"`
}

func (c *Client) SendMessage(ctx context.Context, chatID string, text string, replyToMsgID int) (int64, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	var parsedChatID interface{} = chatID
	if idInt, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		parsedChatID = idInt
	}

	payload := SendMessageRequest{
		ChatID:           parsedChatID,
		Text:             text,
		ReplyToMessageID: replyToMsgID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("sendMessage error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res TelegramResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return 0, err
	}
	if !res.Ok {
		return 0, fmt.Errorf("telegram send error: %s", res.Description)
	}

	var msgRes MessageResult
	if err := json.Unmarshal(res.Result, &msgRes); err != nil {
		return 0, err
	}

	return msgRes.MessageID, nil
}
