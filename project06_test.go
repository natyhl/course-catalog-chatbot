package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

var testDB *DB

func TestMain(m *testing.M) {
	if os.Getenv("OPENAI_PROJECT_KEY") == "" {
		fmt.Println("OPENAI_PROJECT_KEY is not set")
		os.Exit(1)
	}

	testDB = NewDB()
	var count int
	// Check how many courses are already in the database and put the result in count
	// --> Built db if not already existing
	if err := testDB.db.QueryRow("SELECT COUNT(*) FROM course").Scan(&count); err != nil {
		fmt.Printf("count query failed: %v\n", err)
		os.Exit(1)
	}
	if count == 0 {
		load_csv(testDB)
	}

	exitCode := m.Run() // Run tests

	// Close the database and exit
	testDB.db.Close()
	os.Exit(exitCode)
}

// Check API availability
func requireAPIKey(t *testing.T) {
	if os.Getenv("OPENAI_PROJECT_KEY") == "" {
		t.Fatalf("OPENAI_PROJECT_KEY is not set up")
	}
}

// Build/reuse db
func newTestDB(t *testing.T) *DB {
	return testDB
}

func ask(t *testing.T, db *DB, q string) string {
	return db.Chat(q)
}

func judgeSimilar(t *testing.T, db *DB, want, got string) int {
	systemPrompt := `You are an analyst for AI evaluation.
Score how similar the answer is to the expected answer, on a scale of 1–3:
1 = Low similarity (wrong or off-topic)
2 = Medium similarity
3 = High similarity (correct and complete)

Respond ONLY with the number 1, 2, or 3.`
	userPrompt := "Expected answer:\n" + want + "\n\nChat answer:\n" + got

	text, _, err := db.Chatbot(systemPrompt, userPrompt)
	if err != nil {
		t.Fatalf("judgeSimilar API call failed: %v", err)
	}
	switch text {
	case "3":
		return 3
	case "2":
		return 2
	case "1":
		return 1
	default:
		t.Fatalf("invalid judge output: %q", text)
		return 0
	}
}

type TestCase struct {
	Name     string
	Question string
	Want     string
	MinScore int
}

func TestProject06(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	tests := []TestCase{
		{
			Name:     "TestCS272",
			Question: "Who is teaching CS 272?",
			Want:     "Philip Peterson teaches CS 272.",
			MinScore: 2,
		},
		{
			Name:     "TestEmail",
			Question: "What's his email address?",
			Want:     "Philip Peterson's email address is phpeterson@usfca.edu",
			MinScore: 2,
		},
		{
			Name:     "TestPhilGreg",
			Question: "What are Phil Peterson and Greg Benson teaching?",
			Want:     "Phil Peterson teaches Software Development (CS 272) and Greg Benson teaches Operating Systems (CS 315).",
			MinScore: 2,
		},
		{
			Name:     "TestHR148",
			Question: "Which courses is Phil Peterson teaching in HR 148?",
			Want:     "Phil Peterson teaches Software Development (CS 272) in HR 148.",
			MinScore: 2,
		},
		{
			Name:     "TestKHall",
			Question: "Which department's courses are most frequently scheduled in Kalmanovitz (KA) Hall?",
			Want:     "Provide the department(s) with the most courses in Kalmanovitz Hall.",
			MinScore: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Reset dialogue for each test to avoid context contamination
			db.dialogue = []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a assistant for USF course queries. Use search_courses to find information. Answer in 1-2 plain sentences without any formatting, bullet points, numbered lists, bold text, or line breaks. Use format like: 'Course X and Course Y are offered' or 'Professor teaches Course A (code) and Course B (code)'.",
				},
			}

			got := ask(t, db, tc.Question)
			score := judgeSimilar(t, db, tc.Want, got)
			t.Logf("[%s] judge similarity score = %d", tc.Name, score)
			if score < tc.MinScore {
				t.Fatalf("%s failed: similarity %d < %d\nQuestion: %q\nGot: %q\nWant: %q",
					tc.Name, score, tc.MinScore, tc.Question, got, tc.Want)
			}
		})
	}
}
