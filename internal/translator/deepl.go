package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// deeplClient calls the DeepL translation API.
type deeplClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewDeepLClient creates a DeepL translator. Set pro=true for paid API endpoint.
func NewDeepLClient(apiKey string, pro bool) Translator {
	baseURL := "https://api-free.deepl.com/v2"
	if pro {
		baseURL = "https://api.deepl.com/v2"
	}
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &deeplClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

type deeplRequest struct {
	Text       []string `json:"text"`
	SourceLang string   `json:"source_lang"`
	TargetLang string   `json:"target_lang"`
}

type deeplResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
	Message string `json:"message,omitempty"` // error message
}

// normLang normalises language codes for DeepL:
// "EN" → "EN-US" when used as target (DeepL requires regional variant).
func normLang(lang, role string) string {
	lang = strings.ToUpper(lang)
	if role == "target" && lang == "EN" {
		return "EN-US"
	}
	return lang
}

func (d *deeplClient) Translate(ctx context.Context, text, srcLang, dstLang string) (string, error) {
	body := deeplRequest{
		Text:       []string{text},
		SourceLang: normLang(srcLang, "source"),
		TargetLang: normLang(dstLang, "target"),
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("deepl: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		d.baseURL+"/translate", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("deepl: build request: %w", err)
	}
	req.Header.Set("Authorization", "DeepL-Auth-Key "+d.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepl: http: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode == 403 {
		return "", fmt.Errorf("deepl: authentication failed (403) — check DEEPL_API_KEY")
	}
	if resp.StatusCode == 429 {
		return "", fmt.Errorf("deepl: rate limit exceeded (429)")
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("deepl: server error (%d)", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("deepl: read body: %w", err)
	}

	var dr deeplResponse
	if err := json.Unmarshal(raw, &dr); err != nil {
		return "", fmt.Errorf("deepl: parse response: %w", err)
	}
	if dr.Message != "" {
		return "", fmt.Errorf("deepl: api error: %s", dr.Message)
	}
	if len(dr.Translations) == 0 {
		return "", fmt.Errorf("deepl: empty translations in response")
	}

	return strings.TrimSpace(dr.Translations[0].Text), nil
}
