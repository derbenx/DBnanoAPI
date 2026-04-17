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

	ModelNanoFlash   string `json:"model_nano_flash"`
	ModelNanoPro     string `json:"model_nano_pro"`
	ModelNano2       string `json:"model_nano_2"`
	ModelImagen      string `json:"model_imagen"`
	ModelImagenUltra string `json:"model_imagen_ultra"`

	// GUI State
	WindowWidth  float32 `json:"window_width"`
	WindowHeight float32 `json:"window_height"`
	IsMaximized  bool    `json:"is_maximized"`

	SplitOffsetMain float64 `json:"split_offset_main"`
	SplitOffsetLeft float64 `json:"split_offset_left"`
	SplitOffsetTop  float64 `json:"split_offset_top"`
	LogSplitOffset  float64 `json:"log_split_offset"`
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
	RunningCount   int
}

type BatchJob struct {
	JobID       string
	Status      string
	SubmittedAt time.Time
	Progress    string
}
