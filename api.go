package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type GeminiRequest struct {
	Contents          []Content         `json:"contents"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	SafetySettings    []SafetySetting   `json:"safetySettings"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GenerationConfig struct {
	CandidateCount     int          `json:"candidateCount"`
	ResponseModalities []string     `json:"responseModalities"`
	ImageConfig        *ImageConfig `json:"imageConfig,omitempty"`
}

type ImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

var ModelMapping = map[string]string{
	"Nano Flash":   "gemini-2.5-flash-image",
	"Nano Pro":     "gemini-3-pro-image-preview",
	"Nano 2":       "gemini-3.1-flash-image-preview",
	"Imagen":       "imagen-4.0-generate-001",
	"Imagen Ultra": "imagen-4.0-ultra-generate-001",
}

func (s *AppState) RunTask(task *TaskInfo) error {
	modelID := ModelMapping[task.Agent]
	if modelID == "" {
		modelID = task.Agent // Fallback to raw if not found
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, s.Config.APIKey)

	reqBody, err := s.BuildPayload(task)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return s.HandleError(body, resp.StatusCode)
	}

	return s.ProcessResponse(body, task)
}

func (s *AppState) BuildPayload(task *TaskInfo) ([]byte, error) {
	fullPrompt := fmt.Sprintf("USER DIRECTIVE: %s. Aspect Ratio: %s. Avoid: %s", task.Prompt, task.Ratio, task.NegativePrompt)
	parts := []Part{{Text: fullPrompt}}
	encourage := s.Config.EncourageGen

	if task.SourcePath != "" && task.SourcePath != "<GENERATE>" {
		encourage = s.Config.EncourageEdt
		paths := strings.Split(task.SourcePath, "|")
		for _, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			b64 := base64.StdEncoding.EncodeToString(data)
			parts = append(parts, Part{
				InlineData: &InlineData{
					MimeType: "image/jpeg", // Simplified
					Data:     b64,
				},
			})
		}
	}

	req := GeminiRequest{
		Contents: []Content{{Parts: parts}},
		SystemInstruction: &Content{Parts: []Part{{
			Text: encourage,
		}}},
		SafetySettings: s.Config.SafetySettings,
		GenerationConfig: &GenerationConfig{
			CandidateCount:     1,
			ResponseModalities: []string{"IMAGE"},
			ImageConfig: &ImageConfig{
				AspectRatio: task.Ratio,
				ImageSize:   task.Size,
			},
		},
	}

	return json.Marshal(req)
}

func (s *AppState) HandleError(body []byte, status int) error {
	// Ported InspectApiResponse logic
	bodyStr := string(body)
	if strings.Contains(bodyStr, "<html") {
		re := regexp.MustCompile(`(?i)<h1>(.*?)</h1>`)
		match := re.FindStringSubmatch(bodyStr)
		if len(match) > 1 {
			return fmt.Errorf("HTML Error: %s", match[1])
		}
		return fmt.Errorf("HTTP Error %d (HTML response)", status)
	}

	// Handle structured JSON error
	type ErrorDetail struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	type ErrorResp struct {
		Error ErrorDetail `json:"error"`
	}

	var errResp ErrorResp
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("API Error %d (%s): %s", errResp.Error.Code, errResp.Error.Status, errResp.Error.Message)
	}

	// Handle array-wrapped error (user example)
	var errArray []ErrorResp
	if err := json.Unmarshal(body, &errArray); err == nil && len(errArray) > 0 && errArray[0].Error.Message != "" {
		return fmt.Errorf("API Error %d (%s): %s", errArray[0].Error.Code, errArray[0].Error.Status, errArray[0].Error.Message)
	}

	return fmt.Errorf("HTTP Error %d: %s", status, bodyStr)
}

func (s *AppState) TestAPI() error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", s.Config.APIKey)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return s.HandleError(body, resp.StatusCode)
	}

	var modelsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return err
	}

	s.Log(fmt.Sprintf("Connection successful. Found %d models.", len(modelsResp.Models)))
	// Log first 10 models to avoid log spamming if there are many
	limit := 10
	if len(modelsResp.Models) < limit {
		limit = len(modelsResp.Models)
	}
	for i := 0; i < limit; i++ {
		s.Log(" - " + modelsResp.Models[i].Name)
	}
	if len(modelsResp.Models) > limit {
		s.Log(fmt.Sprintf(" ... and %d more.", len(modelsResp.Models)-limit))
	}
	return nil
}

func (s *AppState) ProcessResponse(body []byte, task *TaskInfo) error {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	if len(resp.Candidates) == 0 {
		return fmt.Errorf("no candidates in response")
	}

	cand := resp.Candidates[0]
	if cand.FinishReason != "" && cand.FinishReason != "STOP" && cand.FinishReason != "SUCCESS" {
		return fmt.Errorf("finish reason: %s", cand.FinishReason)
	}

	for _, part := range cand.Content.Parts {
		if part.InlineData.Data != "" {
			data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				return err
			}

			ext := "jpg"
			if strings.Contains(strings.ToLower(part.InlineData.MimeType), "png") {
				ext = "png"
			}

			fileName := fmt.Sprintf("GoTask_%d_%d.%s", task.ID, time.Now().Unix(), ext)
			outPath := filepath.Join(s.Config.OutputDir, fileName)

			os.MkdirAll(s.Config.OutputDir, 0755)
			return os.WriteFile(outPath, data, 0644)
		}
	}

	return fmt.Errorf("no image data in response")
}
