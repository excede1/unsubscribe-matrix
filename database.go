package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// initDatabase initializes the SQLite database and creates the table if it doesn't exist
func initDatabase() error {
	var err error

	// Open SQLite database (creates file if it doesn't exist)
	db, err = sql.Open("sqlite3", "./email_processing.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create the email_processing_records table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS email_processing_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		email TEXT NOT NULL,
		action TEXT NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// closeDatabase closes the database connection
func closeDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// insertEmailProcessingRecord inserts a new email processing record into the database
func insertEmailProcessingRecord(email, action string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Get current time in Sydney timezone
	sydneyLocation, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		log.Printf("WARNING: Failed to load Sydney timezone, using UTC: %v", err)
		sydneyLocation = time.UTC
	}

	timestamp := time.Now().In(sydneyLocation)

	// Map the action to the correct database format
	var dbAction string
	switch action {
	case "pause":
		dbAction = "PAUSE"
	case "international":
		dbAction = "BBAU"
	case "unsubscribe":
		dbAction = "UNSUBSCRIBE"
	case "subscription_update":
		dbAction = "SUBSCRIPTION_UPDATE"
	case "unsubscribe_all":
		dbAction = "UNSUBSCRIBE_ALL"
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	insertSQL := `
	INSERT INTO email_processing_records (timestamp, email, action)
	VALUES (?, ?, ?)`

	_, err = db.Exec(insertSQL, timestamp, email, dbAction)
	if err != nil {
		return fmt.Errorf("failed to insert email processing record: %w", err)
	}

	log.Printf("Database: Successfully recorded %s action for email %s at %s", dbAction, email, timestamp.Format("2006-01-02 15:04:05 MST"))
	return nil
}

// getEmailProcessingRecords retrieves all email processing records from the database
// This function is provided for future use (e.g., for a results page)
func getEmailProcessingRecords() ([]EmailProcessingRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT id, timestamp, email, action
	FROM email_processing_records
	ORDER BY timestamp DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query email processing records: %w", err)
	}
	defer rows.Close()

	var records []EmailProcessingRecord
	for rows.Next() {
		var record EmailProcessingRecord
		var timestampStr string

		err := rows.Scan(&record.ID, &timestampStr, &record.Email, &record.Action)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse the timestamp
		record.Timestamp, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", timestampStr)
		if err != nil {
			// Try alternative format
			record.Timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr)
			if err != nil {
				log.Printf("WARNING: Failed to parse timestamp %s: %v", timestampStr, err)
				record.Timestamp = time.Now()
			}
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

// EmailProcessingRecord represents a record in the email_processing_records table
type EmailProcessingRecord struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Email     string    `json:"email"`
	Action    string    `json:"action"`
}

// getActionSummary retrieves summary counts for each action type
func getActionSummary() (map[string]int, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT action, COUNT(*) as count
	FROM email_processing_records
	GROUP BY action`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query action summary: %w", err)
	}
	defer rows.Close()

	summary := make(map[string]int)
	for rows.Next() {
		var action string
		var count int

		err := rows.Scan(&action, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary row: %w", err)
		}

		summary[action] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating summary rows: %w", err)
	}

	return summary, nil
}

// getAllRecordsForDisplay retrieves all records formatted for display with Sydney timezone
func getAllRecordsForDisplay() ([]DisplayRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT timestamp, email, action
	FROM email_processing_records
	ORDER BY timestamp DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query records for display: %w", err)
	}
	defer rows.Close()

	sydneyLocation, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		log.Printf("WARNING: Failed to load Sydney timezone, using UTC: %v", err)
		sydneyLocation = time.UTC
	}

	var records []DisplayRecord
	for rows.Next() {
		var record DisplayRecord
		var timestampStr string

		err := rows.Scan(&timestampStr, &record.Email, &record.Action)
		if err != nil {
			return nil, fmt.Errorf("failed to scan display row: %w", err)
		}

		// Parse the timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", timestampStr)
		if err != nil {
			// Try alternative format
			timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr)
			if err != nil {
				log.Printf("WARNING: Failed to parse timestamp %s: %v", timestampStr, err)
				timestamp = time.Now()
			}
		}

		// Convert to Sydney timezone and format for display
		sydneyTime := timestamp.In(sydneyLocation)
		record.FormattedDate = sydneyTime.Format("2006-01-02 15:04:05 MST")

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating display rows: %w", err)
	}

	return records, nil
}

// DisplayRecord represents a record formatted for display
type DisplayRecord struct {
	FormattedDate string `json:"formatted_date"`
	Email         string `json:"email"`
	Action        string `json:"action"`
}

// clearAllRecords deletes all records from the email_processing_records table
func clearAllRecords() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	deleteSQL := `DELETE FROM email_processing_records`

	result, err := db.Exec(deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to clear records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("WARNING: Could not get rows affected count: %v", err)
	} else {
		log.Printf("Successfully cleared %d records from database", rowsAffected)
	}

	return nil
}

// getRecordsByAction retrieves records filtered by action type for CSV export
func getRecordsByAction(action string) ([]DisplayRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT timestamp, email, action
	FROM email_processing_records
	WHERE action = ?
	ORDER BY timestamp DESC`

	rows, err := db.Query(query, action)
	if err != nil {
		return nil, fmt.Errorf("failed to query records by action: %w", err)
	}
	defer rows.Close()

	sydneyLocation, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		log.Printf("WARNING: Failed to load Sydney timezone, using UTC: %v", err)
		sydneyLocation = time.UTC
	}

	var records []DisplayRecord
	for rows.Next() {
		var record DisplayRecord
		var timestampStr string

		err := rows.Scan(&timestampStr, &record.Email, &record.Action)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record row: %w", err)
		}

		// Parse the timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", timestampStr)
		if err != nil {
			// Try alternative format
			timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr)
			if err != nil {
				log.Printf("WARNING: Failed to parse timestamp %s: %v", timestampStr, err)
				timestamp = time.Now()
			}
		}

		// Convert to Sydney timezone and format for display
		sydneyTime := timestamp.In(sydneyLocation)
		record.FormattedDate = sydneyTime.Format("2006-01-02 15:04:05 MST")

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating record rows: %w", err)
	}

	return records, nil
}
