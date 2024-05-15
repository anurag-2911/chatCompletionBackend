package app

import (
	"bytes"
	"chatcompletion/app/chatcompletion/model"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rs/cors"
)

const (
	apiURL = "https://api.openai.com/v1/chat/completions"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func CallOpenAI() {
	mux := http.NewServeMux()
	mux.HandleFunc("/call-openai", handler)
	mux.HandleFunc("/get-exercise-data", getExerciseDataHandler)
	// http.HandleFunc("/call-openai", handler)
	// Setup CORS : for local testing
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Allowing the frontend server
		AllowedMethods:   []string{"POST", "GET"},           // Methods allowed for the CORS-enabled resources
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            true, // Enable debug to see what's happening
	})

	// Wrap your HTTP handler with the CORS handler
	handler := c.Handler(mux)
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))

}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "API key not set in environment variables", http.StatusInternalServerError)
		return
	}

	var userInput struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		{
			Role:    "user",
			Content: userInput.Content,
		},
	}

	chatReq := struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
	}{
		Model:    "gpt-4",
		Messages: messages,
	}

	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling request: %v", err), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating request: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making request: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var openAIResp model.OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding response body: %v", err), http.StatusInternalServerError)
		return
	}

	if len(openAIResp.Choices) > 0 && len(openAIResp.Choices[0].Message.Content) > 0 {
		fmt.Fprint(w, openAIResp.Choices[0].Message.Content)
	} else {
		http.Error(w, "No content found in response", http.StatusInternalServerError)
	}
}

// ================================
// BRIAN'S MODIFICATIONS BEGIN HERE

// Message struct to hold the role and content of the message
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Exercise struct to hold each exercise entry
type Exercise struct {
	Day  string `json:"day"`
	Name string `json:"name"`
	Sets int    `json:"sets"`
	Reps string `json:"reps"`
}

// validateCSV checks if the CSV string is properly formatted
func validateCSV(records [][]string) error {
	for i, record := range records {
		if len(record) != 4 {
			return fmt.Errorf("record on line %d is malformed: %v", i+1, record)
		}
	}
	return nil
}

// parseExercises parses the CSV records into a slice of Exercise structs
func parseExercises(records [][]string) ([]Exercise, error) {
	var exercises []Exercise

	for _, record := range records {
		exercise := Exercise{
			Day:  record[0],
			Name: record[1],
		}
		fmt.Sscanf(record[2], "%d", &exercise.Sets)
		exercise.Reps = record[3]

		exercises = append(exercises, exercise)
	}

	return exercises, nil
}

func handleExerciseData(csvString string) ([]Exercise, error) {

	// example csvString:
	// 	csvString := `Monday,Squats,3,12 reps
	// Tuesday,Push-ups,3,10 reps
	// Wednesday,Plank,3,30 seconds
	// Thursday,Lunges,3,10 reps per leg
	// Friday,Mountain Climbers,3,20 reps
	// Saturday,Deadlifts,3,10 reps
	// Sunday,Burpees,3,10 reps`

	// Read the CSV string
	r := csv.NewReader(strings.NewReader(csvString))
	records, err := r.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV: %v\n", err)
		return nil, err
	}

	// Validate the CSV
	err = validateCSV(records)
	if err != nil {
		fmt.Printf("CSV validation error: %v\n", err)
		return nil, err
	}

	// Parse the records into Exercise structs
	exercises, err := parseExercises(records)
	if err != nil {
		fmt.Printf("Error parsing exercises: %v\n", err)
		return nil, err
	}

	// Return the parsed exercises
	return exercises, nil
}

// getExerciseDataHandler handles the POST request to get exercise data
func getExerciseDataHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "API key not set in environment variables", http.StatusInternalServerError)
		return
	}

	// if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
	// 	http.Error(w, "Error decoding request body", http.StatusBadRequest)
	// 	return
	// }

	messages := []Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		{
			Role: "user",
			Content: `Give me a one exercise for each of the 7 days of the week. Format each exercise and day on a new line, with each line formatted as comma separated values as follows:
day,exercise,sets,reps/duration with unit (seconds, reps, per leg, etc.)
Do not include rest days. Include no additional text in your response.`,
		},
	}

	chatReq := chatRequest{
		Model:    "gpt-4o",
		Messages: messages,
	}

	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling request: %v", err), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating request: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making request: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var openAIResp model.OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding response body: %v", err), http.StatusInternalServerError)
		return
	}

	if !(len(openAIResp.Choices) > 0 && len(openAIResp.Choices[0].Message.Content) > 0) {
		http.Error(w, "No content found in response", http.StatusInternalServerError)
	}

	respData := openAIResp.Choices[0].Message.Content
	parsedData, err := handleExerciseData(respData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error handling exercise data: %v", err), http.StatusInternalServerError)
		return
	}

	// Marshal the exercises to JSON
	jsonData, err := json.Marshal(parsedData)
	if err != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
