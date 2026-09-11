package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func decimalFrom(t *testing.T, s string) decimal.Decimal {
	t.Helper()

	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

func TestTelegramGetUpdates(t *testing.T) {
	var gotPath, gotOffset string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotOffset = r.Form.Get("offset")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":[
			{"update_id":11,"message":{"message_id":1,"from":{"id":7,"first_name":"Tom"},"chat":{"id":7},"text":"-5000 еда"}}
		]}`))
	}))
	defer server.Close()

	api := NewTelegram("token")
	api.baseURL = server.URL

	updates, err := api.GetUpdates(context.Background(), 11, time.Second)
	if err != nil {
		t.Fatalf("get updates: %v", err)
	}

	if gotPath != "/bottoken/getUpdates" {
		t.Errorf("path = %q", gotPath)
	}
	if gotOffset != "11" {
		t.Errorf("offset = %q, want 11", gotOffset)
	}
	if len(updates) != 1 || updates[0].Message.Text != "-5000 еда" {
		t.Fatalf("updates = %+v", updates)
	}
	if updates[0].Message.From.Name() != "Tom" {
		t.Errorf("name = %q", updates[0].Message.From.Name())
	}
}

func TestTelegramReportsRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer server.Close()

	api := NewTelegram("token")
	api.baseURL = server.URL

	if err := api.Send(context.Background(), 7, "hello"); err == nil {
		t.Fatal("a refusal was reported as success")
	}
}

func TestUserName(t *testing.T) {
	cases := map[string]struct {
		user *User
		want string
	}{
		"first name": {&User{ID: 1, FirstName: "Tom", Username: "tomq"}, "Tom"},
		"handle":     {&User{ID: 1, Username: "tomq"}, "@tomq"},
		"id":         {&User{ID: 42}, "42"},
		"nobody":     {nil, "unknown"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := c.user.Name(); got != c.want {
				t.Errorf("name = %q, want %q", got, c.want)
			}
		})
	}
}
