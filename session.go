package main

import (
	"encoding/json"
	"io"
)

type SessionData struct {
	Images []*ImageInfo `json:"images"`
	Tasks  []*TaskInfo  `json:"tasks"`
}

func (s *AppState) SaveSession(w io.Writer) error {
	data := SessionData{
		Images: s.Images,
		Tasks:  s.Tasks,
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}

func (s *AppState) LoadSession(r io.Reader) error {
	bytes, err := io.ReadAll(r)
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
