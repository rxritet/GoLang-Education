package main

import (
	"encoding/json"
	"os"
)

// load reads todos from a JSON file at path.
// If the file does not exist, it returns an empty Store and no error.
func load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{}, nil
		}
		return nil, err
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store, nil
}

// save writes todos to a JSON file at path with indentation.
func save(path string, s Store) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
