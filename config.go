package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

func GetConfigPath() string {
	execPath, _ := os.Executable()
	return filepath.Join(filepath.Dir(execPath), configFileName)
}

func LoadConfig() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func DefaultConfig() *Config {
	return &Config{
		OutputDir:    "img",
		EncourageEdt: "You are a professional image-restoration engine. Please apply the 'USER DIRECTIVE' while maintaining strict structural integrity. Focus on high-fidelity surface rendering and cinematic lighting. Ensure all facial features are sharp, clear and perfectly aligned with the reference without looking plastic. Resolve blur into crisp, clean, 8k-resolution details. Maintain 100% adherence to the subject's identity. If the directive involves clothing, ensure the new attire is rendered with realistic fabric textures and consistent coverage.",
		EncourageGen: "You are a world-class visual concept artist. Please transform the user's prompt into a vivid, high-fidelity masterpiece. Prioritize cinematic lighting, photorealistic textures, and perfect anatomical detail. Every output must be rendered with the clarity of an 8k digital sensor. Interpret abstract concepts as concrete, visually dense scenes. Ensure all subjects, especially faces and hands, are rendered with sharp focus and professional-grade definition.",
		SafetySettings: []SafetySetting{
			{"HARM_CATEGORY_HARASSMENT", "BLOCK_NONE"},
			{"HARM_CATEGORY_HATE_SPEECH", "BLOCK_NONE"},
			{"HARM_CATEGORY_SEXUALLY_EXPLICIT", "BLOCK_NONE"},
			{"HARM_CATEGORY_DANGEROUS_CONTENT", "BLOCK_NONE"},
		},
	}
}
