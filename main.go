// main.go

// Sets up database, loads course data if needed, and runs an interactive
// loop where users can ask questions about USF courses.

package main

import (
	"bufio"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	apiKey := os.Getenv("OPENAI_PROJECT_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_PROJECT_KEY not set")
		os.Exit(1)
	}

	db := NewDB()

	// Only load CSV if database is empty
	var count int
	db.db.QueryRow("SELECT COUNT(*) FROM course").Scan(&count)
	if count == 0 {
		fmt.Println("Database is empty, loading courses...")
		load_csv(db)
	} else {
		fmt.Printf("Database already loaded with %d courses\n", count)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Ask about courses: ")

	// Scan() runs until CTRL-D
	for scanner.Scan() {
		question := scanner.Text()
		if question == "" {
			fmt.Print("Ask about courses: ")
			continue
		}

		answer := db.Chat(question)
		fmt.Println(answer)
		fmt.Print("\nAsk about courses: ")
	}
}
