package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// runREPL starts an interactive command loop, persisting changes after each command.
func runREPL() {
	store, err := load(dataFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading todos:", err)
		os.Exit(1)
	}

	fmt.Println("Todo CLI — interactive mode (type 'help' for commands, 'exit' to quit)")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("todo> ")
		if !scanner.Scan() {
			// EOF (Ctrl+D / Ctrl+Z) — graceful exit
			fmt.Println("\nBye!")
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if done := handleREPLCommand(&store, line); done {
			break
		}
	}
}

// handleREPLCommand dispatches a single line of input. Returns true when user wants to quit.
func handleREPLCommand(store *Store, line string) bool {
	parts := strings.SplitN(line, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "exit", "quit", "q":
		fmt.Println("Bye!")
		return true

	case "help", "h", "?":
		printREPLHelp()

	case "list", "ls":
		store.Print()

	case "add":
		arg = strings.Trim(arg, `"'`)
		if err := runAdd(store, arg); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return false
		}
		if err := save(dataFile, *store); err != nil {
			fmt.Fprintln(os.Stderr, "Error saving:", err)
		}

	case "done":
		id, err := strconv.Atoi(arg)
		if err != nil || id <= 0 {
			fmt.Fprintln(os.Stderr, "Error: provide a valid numeric ID, e.g.  done 2")
			return false
		}
		if err := runDone(store, id); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return false
		}
		if err := save(dataFile, *store); err != nil {
			fmt.Fprintln(os.Stderr, "Error saving:", err)
		}

	case "delete", "del", "rm":
		id, err := strconv.Atoi(arg)
		if err != nil || id <= 0 {
			fmt.Fprintln(os.Stderr, "Error: provide a valid numeric ID, e.g.  delete 2")
			return false
		}
		if err := runDelete(store, id); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return false
		}
		if err := save(dataFile, *store); err != nil {
			fmt.Fprintln(os.Stderr, "Error saving:", err)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q. Type 'help' for available commands.\n", cmd)
	}

	return false
}

func printREPLHelp() {
	fmt.Println("Commands:")
	fmt.Println("  add <title>   Add a new todo")
	fmt.Println("  list          List all todos")
	fmt.Println("  done <id>     Mark a todo as done")
	fmt.Println("  delete <id>   Delete a todo")
	fmt.Println("  help          Show this help")
	fmt.Println("  exit          Quit the program")
}
