package main

import (
	"fmt"
	"os"
	"testing"
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

func TestLab07(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	tests := []TestCase{
		{
			Name:     "TestPhil",
			Question: "What courses is Phil Peterson teaching in Fall 2024?",
			Want:     "Software Development",
			MinScore: 2,
		},
		{
			Name:     "TestPHIL",
			Question: "Which philosophy courses are offered this semester?",
			Want:     "PHIL 110 (Great Philosophical Questions), and PHIL 240 (Ethics).",
			MinScore: 2,
		},
		{
			Name:     "TestBio",
			Question: "Where does Bioinformatics meet?",
			Want:     "Bioinformatics meets in LM 365 (include meeting days/times when available).",
			MinScore: 2,
		},
		{
			Name:     "TestGuitar",
			Question: "Can I learn guitar this semester?",
			Want:     "Answer whether a Guitar course is offered this semester and include course subject/number/title if available.",
			MinScore: 2,
		},
		{
			Name:     "TestMultiple",
			Question: "I would like to take a Rhetoric course from Phil Choong. What can I take?",
			Want:     "RHET 103 Public Speaking (include section/CRN if available).",
			MinScore: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
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

func TestPhil(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	want := "Phil Peterson is teaching Software Development (include course numbers/titles)."
	got := ask(t, db, "What courses is Phil Peterson teaching in Fall 2024?")

	score := judgeSimilar(t, db, want, got)
	if score < 2 {
		t.Fatalf("TestPhil failed (similarity %d < 2)\nGot: %q\nWant: %q", score, got, want)
	}
}

func TestPHIL(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	want := "Philosophy courses offered include PHIL 110 (Great Philosophical Questions), and PHIL 240 (Ethics)."
	got := ask(t, db, "Which philosophy courses are offered this semester?")

	score := judgeSimilar(t, db, want, got)
	if score < 2 {
		t.Fatalf("TestPHIL failed (similarity %d < 2)\nGot: %q\nWant: %q", score, got, want)
	}
}

func TestBio(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	want := "Bioinformatics meets in LM 365 (include meeting days/times when available)."
	got := ask(t, db, "Where does Bioinformatics meet?")

	score := judgeSimilar(t, db, want, got)
	if score < 2 {
		t.Fatalf("TestBio failed (similarity %d < 2)\nGot: %q\nWant: %q", score, got, want)
	}
}

func TestGuitar(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	want := "Answer whether a Guitar course is offered this semester and include course subject/number/title if available."
	got := ask(t, db, "Can I learn guitar this semester?")

	score := judgeSimilar(t, db, want, got)
	if score < 2 {
		t.Fatalf("TestGuitar failed (similarity %d < 2)\nGot: %q\nWant: %q", score, got, want)
	}
}

func TestMultiple(t *testing.T) {
	requireAPIKey(t)
	db := newTestDB(t)

	want := "Return the RHET course(s) taught by Phil Choong, e.g., RHET 103 Public Speaking (include section/CRN if available)."
	got := ask(t, db, "I would like to take a Rhetoric course from Phil Choong. What can I take?")

	score := judgeSimilar(t, db, want, got)
	if score < 2 {
		t.Fatalf("TestMultiple failed (similarity %d < 2)\nGot: %q\nWant: %q", score, got, want)
	}
}
