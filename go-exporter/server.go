package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Struct for request data
type ExportRequest struct {
	Query      string `json:"query"`
	OutputFile string `json:"output_file,omitempty"`
}

// Load Laravel's .env file (Only if not loaded)
func loadEnv() {
	if os.Getenv("DB_USERNAME") == "" { // Only load if env vars are not already set
		err := godotenv.Load("../.env")
		if err != nil {
			log.Println("Warning: Could not load Laravel .env file")
		}
	}
}

// API Handler for exporting data
func exportDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req ExportRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Validate the Query parameter
	if req.Query == "" {
		http.Error(w, "Missing 'query' parameter", http.StatusBadRequest)
		return
	}

	loadEnv()

	dbUser := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_DATABASE")

	// Database connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database connection failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Validate SQL Connection
	if err := db.Ping(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to database: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure storage directory exists
	storagePath := "../storage/app/exports"
	os.MkdirAll(storagePath, os.ModePerm)

	// Set default output filename if not provided
	if req.OutputFile == "" {
		req.OutputFile = fmt.Sprintf("export_%d.csv", time.Now().Unix())
	}

	// Create output CSV file
	filePath := fmt.Sprintf("%s/%s", storagePath, req.OutputFile)
	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create output file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Execute SQL Query
	rows, err := db.Query(req.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Write column headers
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get column names: %v", err), http.StatusInternalServerError)
		return
	}
	writer.Write(columns)

	// Write CSV rows
	for rows.Next() {
		columnsData := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnsData {
			columnPointers[i] = &columnsData[i]
		}
	
		if err := rows.Scan(columnPointers...); err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
	
		row := make([]string, len(columns))
		for i, col := range columnsData {
			if col != nil {
				switch v := col.(type) {
				case []byte:
					row[i] = string(v) // ✅ Convert byte array to string
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			} else {
				row[i] = "NULL" // Handle NULL values
			}
		}
		writer.Write(row)
	}

	// Check for row iteration errors
	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error reading rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response back
	response := map[string]interface{}{
		"message": "Export successful",
		"file":    filePath,
		"url":     fmt.Sprintf("/storage/exports/%s", req.OutputFile), // Laravel storage path
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Start the Go server
func main() {
	http.HandleFunc("/export", exportDataHandler)
	fmt.Println("Go Exporter API is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
