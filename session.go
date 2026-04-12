package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const sessionFileName = "session.json"

type SessionData struct {
	Images    []*ImageInfo `json:"images"`
	Tasks     []*TaskInfo  `json:"tasks"`
	BatchJobs []*BatchJob  `json:"batch_jobs"`
}

func GetSessionPath() string {
	execPath, _ := os.Executable()
	return filepath.Join(filepath.Dir(execPath), sessionFileName)
}

func (s *AppState) SaveSession() error {
	path := GetSessionPath()
	data := SessionData{
		Images:    s.Images,
		Tasks:     s.Tasks,
		BatchJobs: s.BatchJobs,
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

func (s *AppState) LoadSession() error {
	path := GetSessionPath()
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var data SessionData
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}

	s.Images = data.Images
	s.Tasks = data.Tasks
	s.BatchJobs = data.BatchJobs
	return nil
}
