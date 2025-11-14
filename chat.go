package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func (db *DB) Chatbot(systemPrompt, userPrompt string) (content string, totalTokens int, err error) {
	req := openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
	}

	resp, err := db.client.CreateChatCompletion(context.TODO(), req)
	if err != nil {
		return "", 0, err
	}

	fmt.Printf("Tokens: %d\n", resp.Usage.TotalTokens) // print tokens
	return resp.Choices[0].Message.Content, resp.Usage.TotalTokens, nil
}

// Defines tools that LLM can use
func (db *DB) createTools() []openai.Tool {
	// Use JSONSchema to describe the parameters and their types
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "Search key information for course. Can be an instructor name, course subject/department, building/location or any relevant keyword.",
			},
		},
		Required: []string{"query"},
	}

	// Make a function using those parameters
	f := openai.FunctionDefinition{
		Name:        "search_courses",
		Description: "Search the course database for courses matching the query. Use this to find courses by instructor, subject, location, topic, or any other course attribute.",
		Parameters:  params,
	}

	// Make a tool using the function
	t := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f,
	}

	// Return a slice containing this one tool
	return []openai.Tool{t}
}

// Function that runs the tool
// Takes a ToolCall and returns results as a string
func (db *DB) executeTool(toolCall openai.ToolCall) string {
	// Parse the JSON arguments from the tool call
	var args map[string]interface{}
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args) // source: https://stackoverflow.com/questions/37913601/json-parsing-in-golang
	if err != nil {
		return "Error parsing arguments"
	}

	results := db.Query(args["query"].(string)) // search vector database
	return strings.Join(results, "\n")
}

func (db *DB) Chat(question string) string {
	// Get tool definition
	tools := db.createTools()

	// Add question to dialogue
	db.dialogue = append(db.dialogue, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	// First API call
	req := openai.ChatCompletionRequest{
		Model:    "gpt-4o-mini",
		Messages: db.dialogue,
		Tools:    tools,
	}

	resp, err := db.client.CreateChatCompletion(context.TODO(), req)
	if err != nil {
		log.Fatal(err)
	}

	msg := resp.Choices[0].Message
	db.dialogue = append(db.dialogue, msg)

	if len(msg.ToolCalls) == 0 {
		fmt.Printf("Tokens: %d\n", resp.Usage.TotalTokens)
		return msg.Content
		// If tools were requested, loop through each one
	} else {
		for _, toolCall := range msg.ToolCalls {
			result := db.executeTool(toolCall) // Query the database with the LLM's chosen search term

			newmsg := openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			}

			db.dialogue = append(db.dialogue, newmsg)
		}

		// Second API call
		newreq := openai.ChatCompletionRequest{
			Model:    "gpt-4o-mini",
			Messages: db.dialogue,
			Tools:    tools,
		}

		resp, err = db.client.CreateChatCompletion(context.TODO(), newreq)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Add final response to dialogue
	finalMsg := resp.Choices[0].Message
	db.dialogue = append(db.dialogue, finalMsg)

	fmt.Printf("Tokens: %d\n", resp.Usage.TotalTokens)
	finalContent := resp.Choices[0].Message.Content
	if strings.TrimSpace(finalContent) == "" {
		return "I found some information but couldn't formulate a response. Please try again."
	}

	return finalContent
}
