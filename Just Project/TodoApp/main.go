package main

import (
	"flag"
	"fmt"
	"os"
)

const dataFile = "todos.json"

func main() {
	addFlag := flag.String("add", "", "Add a new todo with the given title")
	listFlag := flag.Bool("list", false, "List all todos")
	doneFlag := flag.Int("done", 0, "Mark a todo as done by ID")
	deleteFlag := flag.Int("delete", 0, "Delete a todo by ID")
	interactiveFlag := flag.Bool("interactive", false, "Start interactive REPL mode")
	flag.BoolVar(interactiveFlag, "i", false, "Start interactive REPL mode (shorthand)")

	flag.Parse()

	// No flags provided — show usage and exit 1
	if !flag.Parsed() || flag.NFlag() == 0 {
		fmt.Fprintln(os.Stderr, "Todo CLI — manage your tasks from the terminal")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  go run . --add \"task title\"   Add a new todo")
		fmt.Fprintln(os.Stderr, "  go run . --list               List all todos")
		fmt.Fprintln(os.Stderr, "  go run . --done <id>          Mark a todo as done")
		fmt.Fprintln(os.Stderr, "  go run . --delete <id>        Delete a todo")
		fmt.Fprintln(os.Stderr, "  go run . --interactive        Start interactive REPL mode")
		os.Exit(1)
	}

	// Interactive REPL — runs until the user types 'exit'
	if *interactiveFlag {
		runREPL()
		return
	}

	store, err := load(dataFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading todos:", err)
		os.Exit(1)
	}

	switch {
	case *addFlag != "":
		if err := runAdd(&store, *addFlag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *listFlag:
		store.Print()
		return
	case *doneFlag != 0:
		if err := runDone(&store, *doneFlag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *deleteFlag != 0:
		if err := runDelete(&store, *deleteFlag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "No valid flag provided. Run with no flags for usage.")
		os.Exit(1)
	}

	if err := save(dataFile, store); err != nil {
		fmt.Fprintln(os.Stderr, "Error saving todos:", err)
		os.Exit(1)
	}
}

func runAdd(store *Store, title string) error {
	if title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	todo := store.Add(title)
	fmt.Printf("Added: [%d] %s\n", todo.ID, todo.Title)
	return nil
}

func runDone(store *Store, id int) error {
	if err := store.Complete(id); err != nil {
		return err
	}
	for _, t := range *store {
		if t.ID == id {
			fmt.Printf("Done: [%d] %s\n", t.ID, t.Title)
			return nil
		}
	}
	return nil
}

func runDelete(store *Store, id int) error {
	// Capture title before deletion for output
	title := ""
	for _, t := range *store {
		if t.ID == id {
			title = t.Title
			break
		}
	}
	if err := store.Delete(id); err != nil {
		return err
	}
	fmt.Printf("Deleted: [%d] %s\n", id, title)
	return nil
}
