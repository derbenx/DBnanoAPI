package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const sessionFileName = "session.json"

type SessionData struct {
	Images []*ImageInfo `json:"images"`
	Tasks  []*TaskInfo  `json:"tasks"`
}

func GetSessionPath() string {
	execPath, _ := os.Executable()
	return filepath.Join(filepath.Dir(execPath), sessionFileName)
}

func (s *AppState) SaveSession(path string) error {
	data := SessionData{
		Images: s.Images,
		Tasks:  s.Tasks,
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *AppState) LoadSession(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data SessionData
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}

	s.Images = data.Images
	s.Tasks = data.Tasks
	return nil
}
