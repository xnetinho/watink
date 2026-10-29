package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()
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
	defer func() { _ = resp.Body.Close() }()

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

type TelegramFile struct {
	FileID       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

type GetFileResponse struct {
	Ok     bool         `json:"ok"`
	Result TelegramFile `json:"result"`
}

func (c *Client) GetFile(ctx context.Context, fileID string) (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", c.token, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("getFile error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res GetFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if !res.Ok {
		return "", fmt.Errorf("telegram getFile not ok")
	}
	return res.Result.FilePath, nil
}

func (c *Client) DownloadFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.token, filePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download file error (status %d)", resp.StatusCode)
	}
	return resp.Body, nil
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
	defer func() { _ = resp.Body.Close() }()

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

type SendMediaRequest struct {
	ChatID           interface{} `json:"chat_id"`
	Caption          string      `json:"caption,omitempty"`
	ReplyToMessageID int         `json:"reply_to_message_id,omitempty"`
}

func (c *Client) SendPhoto(ctx context.Context, chatID string, photoURL string, caption string, replyToMsgID int) (int64, error) {
	return c.sendMedia(ctx, "sendPhoto", "photo", chatID, photoURL, caption, replyToMsgID)
}

func (c *Client) SendDocument(ctx context.Context, chatID string, docURL string, caption string, replyToMsgID int) (int64, error) {
	return c.sendMedia(ctx, "sendDocument", "document", chatID, docURL, caption, replyToMsgID)
}

func (c *Client) SendVideo(ctx context.Context, chatID string, url string, caption string, replyToMsgID int) (int64, error) {
	return c.sendMedia(ctx, "sendVideo", "video", chatID, url, caption, replyToMsgID)
}

func (c *Client) SendAudio(ctx context.Context, chatID string, url string, caption string, replyToMsgID int) (int64, error) {
	return c.sendMedia(ctx, "sendAudio", "audio", chatID, url, caption, replyToMsgID)
}

// SendSticker: o sticker do Telegram não tem legenda.
func (c *Client) SendSticker(ctx context.Context, chatID string, url string, replyToMsgID int) (int64, error) {
	return c.sendMedia(ctx, "sendSticker", "sticker", chatID, url, "", replyToMsgID)
}

// ForwardMessage encaminha de verdade (com o selo "Encaminhada de...") uma mensagem de outro chat do mesmo bot.
func (c *Client) ForwardMessage(ctx context.Context, chatID, fromChatID, messageID string) (int64, error) {
	var to, from interface{} = chatID, fromChatID
	if n, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		to = n
	}
	if n, err := strconv.ParseInt(fromChatID, 10, 64); err == nil {
		from = n
	}
	mid, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("forwardMessage: message_id invalido")
	}
	body, _ := json.Marshal(map[string]interface{}{"chat_id": to, "from_chat_id": from, "message_id": mid})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/forwardMessage", c.token), bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("forwardMessage error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	var res TelegramResponse
	if json.Unmarshal(bodyBytes, &res) != nil || !res.Ok {
		return 0, fmt.Errorf("telegram forwardMessage not ok")
	}
	var out MessageResult
	if err := json.Unmarshal(res.Result, &out); err != nil {
		return 0, err
	}
	return out.MessageID, nil
}

// sendMedia envia por multipart quando a mídia é um arquivo local (/public/media/...): o Telegram não alcança esse
// caminho relativo e rejeita com "URL host is empty". URL externa segue por JSON, que o Telegram baixa sozinho.
func (c *Client) sendMedia(ctx context.Context, method, field, chatID, mediaURL, caption string, replyToMsgID int) (int64, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)

	var req *http.Request
	var err error
	if data, local, rerr := mediastore.ReadLocal(mediaURL); local {
		if rerr != nil {
			return 0, fmt.Errorf("%s: ler midia local: %w", method, rerr)
		}
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("chat_id", chatID)
		_ = mw.WriteField("caption", caption)
		if replyToMsgID != 0 {
			_ = mw.WriteField("reply_to_message_id", strconv.Itoa(replyToMsgID))
		}
		fw, ferr := mw.CreateFormFile(field, path.Base(mediaURL))
		if ferr != nil {
			return 0, ferr
		}
		_, _ = fw.Write(data)
		if err = mw.Close(); err != nil {
			return 0, err
		}
		if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, &buf); err != nil {
			return 0, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
	} else {
		var parsedChatID interface{} = chatID
		if idInt, perr := strconv.ParseInt(chatID, 10, 64); perr == nil {
			parsedChatID = idInt
		}
		body, _ := json.Marshal(map[string]interface{}{
			"chat_id":             parsedChatID,
			field:                 mediaURL,
			"caption":             caption,
			"reply_to_message_id": replyToMsgID,
		})
		if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body)); err != nil {
			return 0, err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%s error (status %d): %s", method, resp.StatusCode, string(bodyBytes))
	}
	var res TelegramResponse
	if json.Unmarshal(bodyBytes, &res) != nil || !res.Ok {
		return 0, fmt.Errorf("telegram %s not ok", method)
	}
	var msgRes MessageResult
	if err := json.Unmarshal(res.Result, &msgRes); err != nil {
		return 0, err
	}
	return msgRes.MessageID, nil
}

// editDate é o instante da edição (edit_date), que ordena a cadeia de versões; zero quando não é edição ou o Telegram não o informou.
func editDate(msg *TelegramMessage, edited bool) time.Time {
	if !edited || msg == nil || msg.EditDate == 0 {
		return time.Time{}
	}
	return time.Unix(msg.EditDate, 0)
}
