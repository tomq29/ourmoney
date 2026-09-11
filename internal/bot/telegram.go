package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The bot speaks to three endpoints, so it speaks to them directly rather than
// through a library that wraps a hundred.
type Telegram struct {
	token   string
	baseURL string
	client  *http.Client
}

func NewTelegram(token string) *Telegram {
	return &Telegram{
		token:   token,
		baseURL: "https://api.telegram.org",
		// Long polling holds the request open, so the timeout has to outlast it.
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID int64 `json:"id"`
}

// Name is what the ledger stores as the member's name: a first name if there
// is one, the @handle otherwise, and the id when Telegram offers neither.
func (u *User) Name() string {
	switch {
	case u == nil:
		return "unknown"
	case u.FirstName != "":
		return u.FirstName
	case u.Username != "":
		return "@" + u.Username
	}
	return strconv.FormatInt(u.ID, 10)
}

func (t *Telegram) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	params := url.Values{
		"offset":          {strconv.FormatInt(offset, 10)},
		"timeout":         {strconv.Itoa(int(timeout.Seconds()))},
		"allowed_updates": {`["message"]`},
	}

	var updates []Update
	if err := t.call(ctx, "getUpdates", params, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (t *Telegram) Send(ctx context.Context, chatID int64, text string) error {
	params := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"HTML"},
	}
	return t.call(ctx, "sendMessage", params, nil)
}

func (t *Telegram) call(ctx context.Context, method string, params url.Values, out any) error {
	endpoint := fmt.Sprintf("%s/bot%s/%s", t.baseURL, t.token, method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	defer resp.Body.Close()

	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("%s: decode response: %w", method, err)
	}
	if !envelope.OK {
		return fmt.Errorf("%s: telegram said no: %s", method, envelope.Description)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("%s: decode result: %w", method, err)
	}
	return nil
}
