package main

import "time"

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type Config struct {
	APIKey           string          `json:"api_key"`
	OutputDir        string          `json:"output_dir"`
	DefaultPrompt    string          `json:"default_prompt"`
	DefaultNegPrompt string          `json:"default_neg_prompt"`
	EncourageEdt     string          `json:"encourage_edt"`
	EncourageGen     string          `json:"encourage_gen"`
	Debug            bool            `json:"debug"`
	SafetySettings   []SafetySetting `json:"safety_settings"`
	Temperature      float32         `json:"temperature"`
	TopP             float32         `json:"top_p"`
	TopK             int             `json:"top_k"`
	MaxOutputTokens  int             `json:"max_output_tokens"`
}

type ImageInfo struct {
	ID        string
	FileName  string
	FullPath  string
	SizeMB    float64
	TaskCount int
	Selected  bool // For checkbox selection
}

type TaskInfo struct {
	ID             int // Internal index
	ImgIDs         string
	Agent          string
	Size           string
	Ratio          string
	Status         string
	Cost           float64
	Prompt         string
	NegativePrompt string
	Format         string
	Disabled       bool
	SourcePath     string
}

type BatchJob struct {
	JobID       string
	Status      string
	SubmittedAt time.Time
	Progress    string
}
