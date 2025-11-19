package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	openai "github.com/sashabaranov/go-openai"
)

type Course struct {
	Subj       string
	Number     string
	Section    string
	Title      string
	Instructor string
	Email      string
	Days       string
	StartTime  string
	EndTime    string
	Building   string
	Room       string
}

type DB struct {
	db            *sql.DB
	client        *openai.Client
	instructorMap map[string][]Course // Map: instructor → courses they teach
	subjectMap    map[string][]Course //Map: subject → courses in that subject
	dialogue      []openai.ChatCompletionMessage
}

func NewDB() *DB {
	// Load the vec0 vector extension
	name := "course.db"

	// Check if database exists
	if _, err := os.Stat(name); err == nil {
		fmt.Println("Using existing database:", name) //print message and keep it
	} else {
		os.Remove(name) // remove any old file
		fmt.Println("Creating new database:", name)
	}

	sqlite_vec.Auto() // Load the vector extension for SQLite
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		log.Fatal(err)
	}

	// Check if table exists
	if _, err := db.Exec(`SELECT 1 FROM course LIMIT 1`); err != nil {
		_, err = db.Exec(`
			CREATE VIRTUAL TABLE IF NOT EXISTS course USING vec0(
				id INTEGER PRIMARY KEY,
				plain TEXT,
				embedding FLOAT[3072]
			);
		`)
		if err != nil {
			log.Fatal(err)
		}
	}

	return &DB{
		db:            db,
		client:        openai.NewClient(os.Getenv("OPENAI_PROJECT_KEY")),
		instructorMap: make(map[string][]Course),
		subjectMap:    make(map[string][]Course),
		dialogue: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a assistant for USF course queries using courses.db. Use search_courses to find information. Answer in 1-2 plain sentences without any formatting, bullet points, numbered lists, bold text, or line breaks. Use format like: 'Course X and Course Y are offered' or 'Professor teaches Course A (code) and Course B (code)'.",
			},
		},
	}
}
