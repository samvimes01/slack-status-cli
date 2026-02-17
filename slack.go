package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const slackProfileURL = "https://slack.com/api/users.profile.set"

type statusProfile struct {
	StatusText       string `json:"status_text"`
	StatusEmoji      string `json:"status_emoji"`
	StatusExpiration int64  `json:"status_expiration"`
}

type profileSetRequest struct {
	Profile statusProfile `json:"profile"`
}

type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func SetStatus(token, text, emoji string, expirationUnix int64) error {
	body := profileSetRequest{
		Profile: statusProfile{
			StatusText:       text,
			StatusEmoji:      emoji,
			StatusExpiration: expirationUnix,
		},
	}
	return doProfileSet(token, body)
}

func ClearStatus(token string) error {
	body := profileSetRequest{
		Profile: statusProfile{
			StatusText:       "",
			StatusEmoji:      "",
			StatusExpiration: 0,
		},
	}
	return doProfileSet(token, body)
}

func doProfileSet(token string, body profileSetRequest) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, slackProfileURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var slackResp slackResponse
	if err := json.Unmarshal(respBody, &slackResp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !slackResp.OK {
		return fmt.Errorf("slack API error: %s", slackResp.Error)
	}

	return nil
}
