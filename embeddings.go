// embeddings.go
// This file handles vector embeddings and course data loading.
// Contains:
// - CreateBlob(): Converts text to embeddings using OpenAI API
// - InsertBlob(): Inserts course data and embeddings into SQLite
// - Query(): Performs search on the vector database
// - load_csv(): Reads courses.csv and populates the database with embeddings

package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/sashabaranov/go-openai"
)

// Converts text (single string or batch) into embedding vectors
// Returns embeddings as byte slices
func (db *DB) CreateBlob(input interface{}) [][]byte {
	var inputs []string

	// Handle both string and []string
	switch v := input.(type) { // https://go.dev/doc/effective_go?utm_source=chatgpt.com#type_switch
	case string:
		inputs = []string{v}
	case []string:
		inputs = v
	default:
		log.Fatal("CreateBlob: input must be string or []string")
	}

	req := openai.EmbeddingRequest{
		Input: inputs,
		Model: openai.LargeEmbedding3,
	}
	resp, err := db.client.CreateEmbeddings(context.TODO(), req) // Call OpenAI API to convert text → vector
	if err != nil {
		log.Fatal(err)
	}

	// Serialize embedding into a byte slice for SQLite
	var results [][]byte
	for _, emb := range resp.Data {
		bts, err := sqlite_vec.SerializeFloat32(emb.Embedding)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, bts)
	}
	return results
}

func (db *DB) InsertBlob(rowid int, p string, b []byte) {
	// Insert the embedding we got from OpenAI into the virtual table
	// rowid: unique ID
	// p: plain text - course info
	// b: embedding blob - vector

	_, err := db.db.Exec(`
	INSERT INTO course (id, plain, embedding) VALUES(?, ?, ?);
	`, rowid, p, b)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *DB) Query(p string) []string {
	blobs := db.CreateBlob(p)
	b := blobs[0]

	// Find courses with embeddings similar to query embedding
	rows, err := db.db.Query(`
		SELECT id, plain, distance FROM course WHERE embedding MATCH ? ORDER BY distance LIMIT 3;
	`, b)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var id int32
		var distance float32
		var plain string
		err = rows.Scan(&id, &plain, &distance)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, plain) // Add plain text to results list
	}
	return results
}

func load_csv(db *DB) {
	file, err := os.Open("courses.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		log.Fatal(err)
	}

	// and build name->index map
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}

	var courses []Course
	var lines []string // hold formatted text strings for embeddings

	for {
		record, err := reader.Read() // read each CSV row
		if err != nil {
			break
		}

		// Ensure required columns exist in this row
		required := []string{
			"SUBJ", "CRSE NUM", "SEC",
			"Title Short Desc",
			"Primary Instructor First Name", "Primary Instructor Last Name",
			"Primary Instructor Email",
			"Meet Days", "Begin Time", "End Time",
			"BLDG", "RM",
		}
		ok := true
		for _, k := range required {
			if _, have := idx[k]; !have || idx[k] >= len(record) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		// Normalize instructor name
		first := strings.TrimSpace(record[idx["Primary Instructor First Name"]])
		last := strings.TrimSpace(record[idx["Primary Instructor Last Name"]])
		instr := strings.TrimSpace(first + " " + last)

		// Create a Course struct
		c := Course{
			Subj:       record[idx["SUBJ"]],
			Number:     record[idx["CRSE NUM"]],
			Section:    record[idx["SEC"]],
			Title:      record[idx["Title Short Desc"]],
			Instructor: instr,
			Email:      record[idx["Primary Instructor Email"]],
			Days:       record[idx["Meet Days"]],
			StartTime:  record[idx["Begin Time"]],
			EndTime:    record[idx["End Time"]],
			Building:   record[idx["BLDG"]],
			Room:       record[idx["RM"]],
		}

		// add this course to the maps
		db.instructorMap[c.Instructor] = append(db.instructorMap[c.Instructor], c)
		db.subjectMap[c.Subj] = append(db.subjectMap[c.Subj], c)

		// Format course data as a labeled string
		line := fmt.Sprintf(
			"SUBJ:%s Number:%s Section:%s Title:%s Instructor:%s Email:%s Days:%s Time:%s-%s Building:%s Room:%s",
			c.Subj, c.Number, c.Section, c.Title,
			c.Instructor, c.Email, c.Days,
			c.StartTime, c.EndTime, c.Building, c.Room,
		)

		courses = append(courses, c)
		lines = append(lines, line)
	}

	total := len(courses)
	fmt.Printf("Loading %d courses into database...\n", total)

	// process 100 courses at a time
	batchSize := 100
	rowid := 1

	// Calculate where this batch ends
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := lines[i:end] // Get a slice of 100 strings
		blobs := db.CreateBlob(batch)

		for j, blob := range blobs {
			db.InsertBlob(rowid, batch[j], blob)
			rowid++
		}

		fmt.Printf("Progress: %d/%d courses loaded\n", end, total)
	}

	fmt.Println("Database loaded successfully!")
}
