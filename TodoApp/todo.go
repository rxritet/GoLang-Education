package main

import (
	"fmt"
	"time"
)

// Todo represents a single task item.
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is a slice of Todo items.
type Store []Todo

// Add creates a new Todo with a monotonically increasing ID.
func (s *Store) Add(title string) Todo {
	maxID := 0
	for _, t := range *s {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	todo := Todo{
		ID:        maxID + 1,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}
	*s = append(*s, todo)
	return todo
}

// Complete marks the Todo with the given ID as done.
func (s *Store) Complete(id int) error {
	for i, t := range *s {
		if t.ID == id {
			(*s)[i].Done = true
			return nil
		}
	}
	return fmt.Errorf("todo %d not found", id)
}

// Delete removes the Todo with the given ID from the store.
func (s *Store) Delete(id int) error {
	for i, t := range *s {
		if t.ID == id {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo %d not found", id)
}

// Print displays all todos in a formatted table.
func (s Store) Print() {
	if len(s) == 0 {
		fmt.Println("No todos yet. Add one with --add")
		return
	}
	fmt.Printf("%-4s  %-6s  %-30s  %s\n", "ID", "Status", "Title", "Created")
	fmt.Printf("%-4s  %-6s  %-30s  %s\n", "----", "------", "------------------------------", "-------------------")
	for _, t := range s {
		status := "[ ]"
		if t.Done {
			status = "[✓]"
		}
		created := t.CreatedAt.Format("2006-01-02 15:04")
		fmt.Printf("%-4d  %-6s  %-30s  %s\n", t.ID, status, t.Title, created)
	}
}
