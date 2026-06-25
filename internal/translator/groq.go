package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// groqClient calls the Groq chat-completions endpoint using HTTP keep-alive.
type groqClient struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

// NewGroqClient creates an optimised Groq API client with connection pooling.
func NewGroqClient(apiKey, model, baseURL string) Translator {
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
	return &groqClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// groqRequest is the JSON body for the chat completions endpoint.
type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
	Error *groqError `json:"error,omitempty"`
}

type groqError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

const groqSystemPrompt = `You are a silent, professional translation engine.
Rules:
- Translate ONLY the user's text. Output nothing else.
- Do NOT add explanations, notes, greetings, or commentary.
- Do NOT wrap the translation in quotes or markdown.
- Preserve formatting: newlines, indentation, code structure.
- If the text is already in the target language, return it unchanged.`

func (g *groqClient) Translate(ctx context.Context, text, srcLang, dstLang string) (string, error) {
	prompt := fmt.Sprintf("Translate the following text from %s to %s:\n\n%s", srcLang, dstLang, text)

	body := groqRequest{
		Model: g.model,
		Messages: []groqMessage{
			{Role: "system", Content: groqSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1, // low temperature → consistent, literal translations
		MaxTokens:   2048,
	}

	return g.doWithRetry(ctx, body, 3)
}

// doWithRetry attempts the API call up to maxRetries times with exponential backoff.
func (g *groqClient) doWithRetry(ctx context.Context, body groqRequest, maxRetries int) (string, error) {
	backoff := []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := g.call(ctx, body)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Don't retry on context cancellation or auth errors.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "invalid_api_key") {
			return "", fmt.Errorf("groq: authentication failed — check GROQ_API_KEY: %w", err)
		}

		if attempt < len(backoff) {
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			wait := backoff[attempt] + jitter
			slog.Warn("groq: retrying", "attempt", attempt+1, "wait", wait, "err", err)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return "", fmt.Errorf("groq: all retries exhausted: %w", lastErr)
}

// call makes a single HTTP request to the Groq API.
func (g *groqClient) call(ctx context.Context, body groqRequest) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("groq: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("groq: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq: http: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) // drain so connection is reused
		resp.Body.Close()
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("groq: read body: %w", err)
	}

	var gr groqResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", fmt.Errorf("groq: parse response: %w", err)
	}

	if gr.Error != nil {
		return "", fmt.Errorf("groq: api error [%s]: %s", gr.Error.Type, gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("groq: empty choices in response")
	}

	result := strings.TrimSpace(gr.Choices[0].Message.Content)
	return result, nil
}
