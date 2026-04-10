package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
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

func (s *AppState) SubmitBatchJob(tasks []*TaskInfo) error {
	if len(tasks) == 0 {
		return nil
	}

	modelName := tasks[0].Agent
	modelID := ModelMapping[modelName]
	if modelID == "" {
		modelID = modelName
	}

	// 1. Create JSONL data
	var buf bytes.Buffer
	for i, t := range tasks {
		payload, err := s.BuildPayload(t)
		if err != nil {
			continue
		}

		// Wrap in Batch format
		// Note: Gemini Batch request structure is slightly different for JSONL
		type BatchReqEntry struct {
			CustomID string          `json:"custom_id"`
			Request  json.RawMessage `json:"request"`
		}

		entry := BatchReqEntry{
			CustomID: fmt.Sprintf("task_%d_%d", t.ID, i),
			Request:  payload,
		}
		line, _ := json.Marshal(entry)
		buf.Write(line)
		buf.WriteString("\n")
	}

	// 2. Upload JSONL to Google Files API
	fileURI, err := s.UploadFile(buf.Bytes())
	if err != nil {
		return fmt.Errorf("upload failed: %v", err)
	}

	// 3. Submit Batch Job
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:batchGenerateContent?key=%s", modelID, s.Config.APIKey)
	submitReq := fmt.Sprintf(`{"batch": {"input_config": {"file_name": "%s"}}}`, fileURI)

	resp, err := http.Post(url, "application/json", strings.NewReader(submitReq))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return s.HandleError(body, resp.StatusCode)
	}

	var res struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return err
	}

	fyne.Do(func() {
		s.BatchJobs = append(s.BatchJobs, &BatchJob{
			JobID:       res.Name,
			Status:      "Submitted",
			SubmittedAt: time.Now(),
			Progress:    "0%",
		})
	})

	s.Log("Batch Job Submitted: " + res.Name)
	return nil
}

func (s *AppState) UploadFile(data []byte) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s", s.Config.APIKey)

	boundary := "NanoGoBoundary" + fmt.Sprint(time.Now().Unix())
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.SetBoundary(boundary)

	// Part 1: Metadata
	metadata := fmt.Sprintf(`{"file": {"display_name": "batch_%d"}}`, time.Now().Unix())
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/json; charset=UTF-8")
	p, _ := writer.CreatePart(h)
	p.Write([]byte(metadata))

	// Part 2: File Content
	h = make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/json")
	p, _ = writer.CreatePart(h)
	p.Write(data)

	writer.Close()

	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("X-Goog-Upload-Protocol", "multipart")
	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)

	var res struct {
		File struct {
			URI string `json:"uri"`
		} `json:"file"`
	}
	json.Unmarshal(resBody, &res)

	if res.File.URI == "" {
		return "", fmt.Errorf("no URI in response: %s", string(resBody))
	}
	return res.File.URI, nil
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

	// Log first 10 models to UI
	limit := 10
	if len(modelsResp.Models) < limit {
		limit = len(modelsResp.Models)
	}
	for i := 0; i < limit; i++ {
		s.Log(" - " + modelsResp.Models[i].Name)
	}

	// Log ALL models to debug.log file unconditionally
	s.LogToFile("Full Model List:")
	for _, m := range modelsResp.Models {
		s.LogToFile(" - " + m.Name)
	}

	if len(modelsResp.Models) > limit {
		s.Log(fmt.Sprintf(" ... and %d more. Full list in debug.log", len(modelsResp.Models)-limit))
	}
	return nil
}

func (s *AppState) CheckBatchStatus(job *BatchJob) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s?key=%s", job.JobID, s.Config.APIKey)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return s.HandleError(body, resp.StatusCode)
	}

	var res struct {
		State         string `json:"state"`
		ResponsesFile string `json:"responsesFile"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return err
	}

	job.Status = res.State
	if res.State == "SUCCEEDED" || res.State == "BATCH_STATE_SUCCEEDED" {
		if res.ResponsesFile != "" {
			s.Log("Batch " + job.JobID + " succeeded. Downloading results...")
			return s.DownloadBatchResults(res.ResponsesFile)
		}
	}
	return nil
}

func (s *AppState) DownloadBatchResults(fileID string) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:download?alt=media&key=%s", fileID, s.Config.APIKey)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// JSONL response with multiple lines
	reader := io.Reader(resp.Body)
	decoder := json.NewDecoder(reader)

	for {
		var result struct {
			CustomID string          `json:"custom_id"`
			Response json.RawMessage `json:"response"`
			Error    json.RawMessage `json:"error"`
		}
		if err := decoder.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if len(result.Error) > 0 {
			s.Log("Task " + result.CustomID + " failed in batch: " + string(result.Error))
			continue
		}

		// Port image extraction logic from ProcessResponse for result.Response
		s.ProcessBatchItem(result.Response, result.CustomID)
	}

	return nil
}

func (s *AppState) ProcessBatchItem(respBody []byte, customID string) {
	// Nested parsing for image data
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
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &resp); err == nil && len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		for _, part := range cand.Content.Parts {
			if part.InlineData.Data != "" {
				data, _ := base64.StdEncoding.DecodeString(part.InlineData.Data)
				ext := "jpg"
				if strings.Contains(strings.ToLower(part.InlineData.MimeType), "png") {
					ext = "png"
				}

				fileName := fmt.Sprintf("Batch_%s_%d.%s", customID, time.Now().Unix(), ext)
				outPath := filepath.Join(s.Config.OutputDir, fileName)

				os.MkdirAll(s.Config.OutputDir, 0755)
				os.WriteFile(outPath, data, 0644)
				s.Log("Saved batch image: " + outPath)
			}
		}
	}
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
			err = os.WriteFile(outPath, data, 0644)
			if err == nil {
				s.Log("Saved image to: " + outPath)
			}
			return err
		}
	}

	return fmt.Errorf("no image data in response")
}
