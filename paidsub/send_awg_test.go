package paidsub

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestSendAWGDocumentUsesFixedFilenameAndRedactsErrors(t *testing.T) {
	const token = "secret-bot-token"
	var filename string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("content type=%q err=%v", mediaType, err)
		}
		reader := multipart.NewReader(request.Body, params["boundary"])
		for {
			part, readErr := reader.NextPart()
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			if part.FormName() == "document" {
				filename = part.FileName()
				data, _ := io.ReadAll(part)
				if string(data) != "full-config" {
					t.Fatalf("document body=%q", data)
				}
			}
		}
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":400,"description":"bad request"}`))}, nil
	})}
	bot := &Bot{client: client, token: token}
	err := bot.sendAWGDocument(context.Background(), 7, 42, []byte("full-config"))
	if filename != "awg-device-42.conf" {
		t.Fatalf("filename=%q", filename)
	}
	if err == nil || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "full-config") {
		t.Fatalf("unsafe error=%v", err)
	}
}

func TestSendAWGDocumentRejectsOversizeWithoutRequest(t *testing.T) {
	called := false
	bot := &Bot{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}, token: "token"}
	if err := bot.sendAWGDocument(context.Background(), 1, 1, make([]byte, maxTelegramAWGDocumentBytes+1)); err == nil {
		t.Fatal("expected oversize error")
	}
	if called {
		t.Fatal("oversize document reached network")
	}
}
