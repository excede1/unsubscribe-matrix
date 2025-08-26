package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

var (
	customerIOSiteID string // Customer.io Site ID for Track API
	customerIOAPIKey string // Customer.io API Key for Track API
	adminUsername    string // Admin username for /results authentication
	adminPassword    string // Admin password for /results authentication
)

// isProduction checks if the application is running in production environment
func isProduction() bool {
	return os.Getenv("FLY_APP_NAME") != ""
}

// isDevelopment checks if the application is running in development environment
func isDevelopment() bool {
	return !isProduction()
}

// setupLogging configures logging based on environment
func setupLogging() error {
	// Set log flags for better debugging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	if isProduction() {
		// In production, log to stdout for fly.io log aggregation
		log.SetOutput(os.Stdout)
		log.Println("Production environment detected - logging to stdout")
		return nil
	}

	// In development, check if LOG_TO_FILE is set
	logToFile := os.Getenv("LOG_TO_FILE")
	if logToFile == "false" {
		// Log to stdout in development if explicitly disabled
		log.SetOutput(os.Stdout)
		log.Println("Development environment - logging to stdout (LOG_TO_FILE=false)")
		return nil
	}

	// Default development behavior - log to file
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("ERROR: Failed to open log file, falling back to stdout: %v", err)
		log.SetOutput(os.Stdout)
		return err
	}

	log.SetOutput(logFile)
	log.Println("Development environment - logging to app.log file")
	return nil
}

// killProcessOnPort kills any existing process on the specified port (development only)
func killProcessOnPort(port string) {
	if isProduction() {
		log.Printf("Production environment - skipping port killing for port %s", port)
		return
	}

	log.Printf("Development environment - checking for existing processes on port %s", port)
	killCmd := exec.Command("lsof", "-ti:"+port)
	if pidBytes, err := killCmd.Output(); err == nil && len(pidBytes) > 0 {
		pidStr := strings.TrimSpace(string(pidBytes))
		if pidStr != "" {
			log.Printf("Found existing process on port %s (PID: %s), killing it...", port, pidStr)
			killProcessCmd := exec.Command("kill", "-9", pidStr)
			if killErr := killProcessCmd.Run(); killErr != nil {
				log.Printf("WARNING: Failed to kill existing process on port %s: %v", port, killErr)
			} else {
				log.Printf("Successfully killed existing process on port %s", port)
				// Give it a moment to fully terminate
				time.Sleep(1 * time.Second)
			}
		}
	} else {
		log.Printf("No existing process found on port %s", port)
	}
}

func main() {
	// Initial log to confirm application start
	log.Println("Application starting...")

	// Detect and log environment
	if isProduction() {
		log.Printf("Running in PRODUCTION environment (FLY_APP_NAME: %s)", os.Getenv("FLY_APP_NAME"))
	} else {
		log.Println("Running in DEVELOPMENT environment")
	}

	// Setup logging based on environment
	if err := setupLogging(); err != nil {
		log.Printf("WARNING: Logging setup encountered an error: %v", err)
	}

	// Load .env file (only in development)
	if isDevelopment() {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file, attempting to use environment-set variables")
		} else {
			log.Println(".env file loaded successfully")
		}
	} else {
		log.Println("Production environment - skipping .env file loading")
	}

	// Load Customer.io Track API credentials
	customerIOSiteID = os.Getenv("CUSTOMERIO_SITE_ID")
	customerIOAPIKey = os.Getenv("CUSTOMERIO_API_KEY")
	if customerIOSiteID == "" {
		log.Fatalln("CRITICAL: CUSTOMERIO_SITE_ID not set in environment variables.")
	}
	if customerIOAPIKey == "" {
		log.Fatalln("CRITICAL: CUSTOMERIO_API_KEY not set in environment variables.")
	}
	log.Println("Customer.io Track API credentials loaded.")

	// Load admin credentials
	adminUsername = os.Getenv("ADMIN_USERNAME")
	adminPassword = os.Getenv("ADMIN_PASSWORD")
	if adminUsername == "" {
		log.Fatalln("CRITICAL: ADMIN_USERNAME not set in environment variables.")
	}
	if adminPassword == "" {
		log.Fatalln("CRITICAL: ADMIN_PASSWORD not set in environment variables.")
	}
	log.Println("Admin credentials loaded.")

	// Initialize database
	if err := initDatabase(); err != nil {
		log.Fatalf("CRITICAL: Failed to initialize database: %v", err)
	}
	log.Println("Database initialization completed.")

	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	log.Println("Fiber app instance created with HTML template engine.")

	// Test route
	app.Get("/ping", func(c *fiber.Ctx) error {
		log.Println("GET /ping request received.")
		return c.SendString("pong")
	})
	log.Println("GET /ping route registered.")

	app.Get("/", func(c *fiber.Ctx) error {
		log.Printf("GET / request received. Path: %s, Query: %s", c.Path(), c.Request().URI().QueryString())
		email := c.Query("email")
		cioID := c.Query("cio")
		action := c.Query("action")
		message := ""
		success := false

		log.Printf("Extracted parameters - Email: '%s', CIO_ID: '%s', Action: '%s'", email, cioID, action)

		// Handle different actions when email is provided
		if email != "" {
			if action != "" {
				log.Printf("Processing action '%s' for email: %s", action, email)

				switch action {
				case "pause":
					err := updateCustomerPausedAttributeByEmail(email)
					if err != nil {
						log.Printf("Error updating 'paused' attribute for email %s: %v", email, err)
						message = "Error processing pause request. Check logs."
					} else {
						message = fmt.Sprintf("Customer (%s) has been paused.", email)
						success = true
						log.Printf("Successfully updated 'paused' attribute for email %s", email)

						// Log to database
						if dbErr := insertEmailProcessingRecord(email, "pause"); dbErr != nil {
							log.Printf("WARNING: Failed to log pause action to database for email %s: %v", email, dbErr)
						}
					}
				case "international":
					err := updateCustomerRelationshipByEmail(email, "BBAU")
					if err != nil {
						log.Printf("Error updating relationship to BBAU for email %s: %v", email, err)
						message = "Error processing international request. Check logs."
					} else {
						message = fmt.Sprintf("Customer (%s) moved to Australian/International list.", email)
						success = true
						log.Printf("Successfully updated relationship to BBAU for email %s", email)

						// Log to database
						if dbErr := insertEmailProcessingRecord(email, "international"); dbErr != nil {
							log.Printf("WARNING: Failed to log international action to database for email %s: %v", email, dbErr)
						}
					}
				case "unsubscribe":
					err := unsubscribeCustomerByEmail(email)
					if err != nil {
						log.Printf("Error unsubscribing email %s: %v", email, err)
						message = "Error processing unsubscribe request. Check logs."
					} else {
						message = fmt.Sprintf("Customer (%s) has been unsubscribed.", email)
						success = true
						log.Printf("Successfully unsubscribed email %s", email)

						// Log to database
						if dbErr := insertEmailProcessingRecord(email, "unsubscribe"); dbErr != nil {
							log.Printf("WARNING: Failed to log unsubscribe action to database for email %s: %v", email, dbErr)
						}
					}
				case "unpause":
					err := updateCustomerUnpausedAttributeByEmail(email)
					if err != nil {
						log.Printf("Error updating 'paused' attribute to false for email %s: %v", email, err)
						message = "Error processing unpause request. Check logs."
					} else {
						message = fmt.Sprintf("Customer (%s) has been unpaused.", email)
						success = true
						log.Printf("Successfully updated 'paused' attribute to false for email %s", email)
					}
				default:
					log.Printf("Unknown action '%s' for email %s", action, email)
					message = "Unknown action requested."
				}
			} else {
				// No action specified, just show the interface
				log.Printf("Email provided (%s) but no action specified. Showing interface.", email)
			}
		} else if cioID != "" {
			// Backward compatibility for customer ID-based requests
			log.Printf("CIO_ID extracted: %s. Using customer ID as identifier.", cioID)

			err := updateCustomerPausedAttribute(cioID)
			if err != nil {
				log.Printf("Error updating 'paused' attribute for cio_id %s: %v", cioID, err)
				message = "Error processing request. Check logs."
			} else {
				message = fmt.Sprintf("Customer (ID: %s) has been paused.", cioID)
				success = true
				log.Printf("Successfully updated 'paused' attribute for cio_id %s. Message: %s", cioID, message)
			}
		}

		if message != "" {
			log.Printf("Message to be displayed: %s. Success: %t", message, success)
		}

		return c.Render("index", fiber.Map{
			"Message": message,
			"Success": success,
			"CioID":   cioID,
			"Action":  action,
		})
	})
	log.Println("GET / route registered.")

	// New subscription management endpoints
	app.Post("/update-subscriptions", handleUpdateSubscriptions)
	log.Println("POST /update-subscriptions route registered.")
	
	app.Post("/unsubscribe-all", handleUnsubscribeAll)
	log.Println("POST /unsubscribe-all route registered.")

	// Protected /results route with authentication
	app.Get("/results", basicAuthMiddleware(adminUsername, adminPassword), handleResults)
	log.Println("GET /results route registered with authentication.")

	// Protected CSV download routes
	app.Get("/results/csv/:action", basicAuthMiddleware(adminUsername, adminPassword), handleCSVDownload)
	log.Println("GET /results/csv/:action route registered with authentication.")

	// Protected clear records route
	app.Post("/results/clear", basicAuthMiddleware(adminUsername, adminPassword), handleClearRecords)
	log.Println("POST /results/clear route registered with authentication.")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Default port if not specified
		log.Println("PORT environment variable not set, using default port 3000.")
	} else {
		log.Printf("PORT environment variable found: %s", port)
	}

	// Kill any existing process on the port before starting (development only)
	killProcessOnPort(port)

	log.Printf("Attempting to start server on port %s...", port)

	// Log startup information based on environment
	if isProduction() {
		log.Printf("Production server starting on port %s", port)
		fmt.Printf("Production server starting on port %s\n", port)
	} else {
		log.Printf("Development server starting on port %s", port)
		fmt.Printf("Development server starting on port %s\n", port)
	}

	// Start server with improved error handling
	errListen := app.Listen(":" + port)
	if errListen != nil {
		// Close database connection before exiting
		if closeErr := closeDatabase(); closeErr != nil {
			log.Printf("WARNING: Failed to close database connection: %v", closeErr)
		}

		if isProduction() {
			log.Fatalf("CRITICAL: Production server failed to start on port %s: %v", port, errListen)
		} else {
			log.Fatalf("CRITICAL: Development server failed to start on port %s: %v", port, errListen)
		}
	}

	// This line would only be reached if Listen() exits gracefully
	log.Println("Server has shut down gracefully.")

	// Close database connection on graceful shutdown
	if closeErr := closeDatabase(); closeErr != nil {
		log.Printf("WARNING: Failed to close database connection: %v", closeErr)
	} else {
		log.Println("Database connection closed successfully.")
	}
}

// updateCustomerPausedAttributeByEmail updates the 'paused' attribute to true using email as identifier via Customer.io Track API.
func updateCustomerPausedAttributeByEmail(email string) error {
	return updateCustomerPausedAttributeFlexible(email, true)
}

// updateCustomerUnpausedAttributeByEmail updates the 'paused' attribute to false using email as identifier via Customer.io Track API.
func updateCustomerUnpausedAttributeByEmail(email string) error {
	return updateCustomerPausedAttributeFlexible(email, false)
}

// updateCustomerPausedAttributeFlexible updates the 'paused' attribute using email as identifier via Customer.io Track API.
func updateCustomerPausedAttributeFlexible(email string, paused bool) error {
	endpointURL := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)

	// Track API uses a simple JSON payload with attributes
	payload := map[string]interface{}{
		"paused": paused,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal Track API payload for email %s: %v", email, err)
		return fmt.Errorf("error marshalling Track API payload: %w", err)
	}

	log.Printf("DEBUG: Attempting to update customer %s via PUT to %s", email, endpointURL)
	log.Printf("DEBUG: Request payload: %s", string(payloadBytes))
	log.Printf("DEBUG: Using Site ID: %s, API Key: %s... (first 10 chars)", customerIOSiteID, customerIOAPIKey[:10])

	req, err := http.NewRequest(http.MethodPut, endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("ERROR: Failed to create Track API request for email %s: %v", email, err)
		return fmt.Errorf("error creating Track API request: %w", err)
	}

	// Track API uses Basic Auth: Site ID as username, API Key as password
	req.SetBasicAuth(customerIOSiteID, customerIOAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CustomerIO-Pauser/1.0")

	log.Printf("DEBUG: Request headers set - Content-Type: application/json, Authorization: Basic [REDACTED]")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send Track API request for email %s: %v", email, err)
		return fmt.Errorf("error sending Track API request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Failed to read Track API response body for email %s: %v", email, readErr)
		// Continue, but log this error.
	}

	log.Printf("DEBUG: Customer.io Track API response for email %s", email)
	log.Printf("DEBUG: Response Status: %s (%d)", resp.Status, resp.StatusCode)
	log.Printf("DEBUG: Response Headers: %v", resp.Header)
	log.Printf("DEBUG: Response Body: %s", string(respBodyBytes))

	// Check if response indicates success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("Customer.io Track API returned non-success status for email %s: %s. Body: %s", email, resp.Status, string(respBodyBytes))
		log.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}

	log.Printf("SUCCESS: Track API request completed for email %s (status %s)", email, resp.Status)
	log.Printf("IMPORTANT: Customer attribute 'paused' should now be visible in Customer.io dashboard")
	log.Printf("  - Using Track API endpoint: %s", endpointURL)
	log.Printf("  - This API directly updates customer profiles in your Customer.io workspace")
	log.Printf("  - If attribute is still not visible, check Customer.io dashboard after 1-2 minutes")

	return nil
}

// updateCustomerRelationshipByEmail manages customer relationships using Customer.io Track API.
// This removes the BBUS relationship and adds the BBAU relationship for international customers.
func updateCustomerRelationshipByEmail(email string, newObjectID string) error {
	log.Printf("DEBUG: Starting relationship update for email %s - removing BBUS and adding %s", email, newObjectID)

	// First, remove the BBUS relationship
	err := removeCustomerRelationship(email, "BBUS")
	if err != nil {
		log.Printf("ERROR: Failed to remove BBUS relationship for email %s: %v", email, err)
		return fmt.Errorf("error removing BBUS relationship: %w", err)
	}

	// Then, add the new relationship (BBAU)
	err = createCustomerRelationship(email, newObjectID)
	if err != nil {
		log.Printf("ERROR: Failed to create %s relationship for email %s: %v", newObjectID, email, err)
		return fmt.Errorf("error creating %s relationship: %w", newObjectID, err)
	}

	log.Printf("SUCCESS: Relationship update completed for email %s - removed BBUS, added %s", email, newObjectID)
	return nil
}

// removeCustomerRelationship removes a relationship between customer and object using Track API
func removeCustomerRelationship(email string, objectID string) error {
	endpointURL := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)

	// Use the delete_relationships action in the customer identification payload
	payload := map[string]interface{}{
		"cio_relationships": map[string]interface{}{
			"action": "delete_relationships",
			"relationships": []map[string]interface{}{
				{
					"identifiers": map[string]interface{}{
						"object_type_id": "1", // Default object type ID
						"object_id":      objectID,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal relationship removal payload for email %s: %v", email, err)
		return fmt.Errorf("error marshalling relationship removal payload: %w", err)
	}

	log.Printf("DEBUG: Attempting to remove relationship %s for customer %s via PUT to %s", objectID, email, endpointURL)
	log.Printf("DEBUG: Request payload: %s", string(payloadBytes))

	req, err := http.NewRequest(http.MethodPut, endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("ERROR: Failed to create relationship removal request for email %s: %v", email, err)
		return fmt.Errorf("error creating relationship removal request: %w", err)
	}

	// Track API uses Basic Auth: Site ID as username, API Key as password
	req.SetBasicAuth(customerIOSiteID, customerIOAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CustomerIO-Pauser/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send relationship removal request for email %s: %v", email, err)
		return fmt.Errorf("error sending relationship removal request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Failed to read relationship removal response body for email %s: %v", email, readErr)
	}

	log.Printf("DEBUG: Relationship removal response for email %s - Status: %s (%d), Body: %s", email, resp.Status, resp.StatusCode, string(respBodyBytes))

	// Check if response indicates success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("Customer.io relationship removal returned non-success status for email %s: %s. Body: %s", email, resp.Status, string(respBodyBytes))
		log.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}

	log.Printf("SUCCESS: Relationship removal completed for email %s and object %s (status %s)", email, objectID, resp.Status)
	return nil
}

// createCustomerRelationship creates a relationship between customer and object using Track API
func createCustomerRelationship(email string, objectID string) error {
	endpointURL := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)

	// Use the add_relationships action in the customer identification payload
	payload := map[string]interface{}{
		"cio_relationships": map[string]interface{}{
			"action": "add_relationships",
			"relationships": []map[string]interface{}{
				{
					"identifiers": map[string]interface{}{
						"object_type_id": "1", // Default object type ID
						"object_id":      objectID,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal relationship creation payload for email %s: %v", email, err)
		return fmt.Errorf("error marshalling relationship creation payload: %w", err)
	}

	log.Printf("DEBUG: Attempting to create relationship %s for customer %s via PUT to %s", objectID, email, endpointURL)
	log.Printf("DEBUG: Request payload: %s", string(payloadBytes))
	log.Printf("DEBUG: Using correct Track API format with cio_relationships and add_relationships action")

	req, err := http.NewRequest(http.MethodPut, endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("ERROR: Failed to create relationship creation request for email %s: %v", email, err)
		return fmt.Errorf("error creating relationship creation request: %w", err)
	}

	// Track API uses Basic Auth: Site ID as username, API Key as password
	req.SetBasicAuth(customerIOSiteID, customerIOAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CustomerIO-Pauser/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send relationship creation request for email %s: %v", email, err)
		return fmt.Errorf("error sending relationship creation request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Failed to read relationship creation response body for email %s: %v", email, readErr)
	}

	log.Printf("DEBUG: Relationship creation response for email %s - Status: %s (%d), Body: %s", email, resp.Status, resp.StatusCode, string(respBodyBytes))

	// Check if response indicates success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("Customer.io relationship creation returned non-success status for email %s: %s. Body: %s", email, resp.Status, string(respBodyBytes))
		log.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}

	log.Printf("SUCCESS: Relationship creation completed for email %s and object %s (status %s)", email, objectID, resp.Status)
	return nil
}

// unsubscribeCustomerByEmail unsubscribes a customer using email as identifier via Customer.io Track API.
func unsubscribeCustomerByEmail(email string) error {
	endpointURL := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)

	// Track API uses a simple JSON payload with attributes
	payload := map[string]interface{}{
		"unsubscribed": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal Track API payload for email %s: %v", email, err)
		return fmt.Errorf("error marshalling Track API payload: %w", err)
	}

	log.Printf("DEBUG: Attempting to unsubscribe customer %s via PUT to %s", email, endpointURL)
	log.Printf("DEBUG: Request payload: %s", string(payloadBytes))
	log.Printf("DEBUG: Using Site ID: %s, API Key: %s... (first 10 chars)", customerIOSiteID, customerIOAPIKey[:10])

	req, err := http.NewRequest(http.MethodPut, endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("ERROR: Failed to create Track API request for email %s: %v", email, err)
		return fmt.Errorf("error creating Track API request: %w", err)
	}

	// Track API uses Basic Auth: Site ID as username, API Key as password
	req.SetBasicAuth(customerIOSiteID, customerIOAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CustomerIO-Pauser/1.0")

	log.Printf("DEBUG: Request headers set - Content-Type: application/json, Authorization: Basic [REDACTED]")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send Track API request for email %s: %v", email, err)
		return fmt.Errorf("error sending Track API request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Failed to read Track API response body for email %s: %v", email, readErr)
		// Continue, but log this error.
	}

	log.Printf("DEBUG: Customer.io Track API response for email %s", email)
	log.Printf("DEBUG: Response Status: %s (%d)", resp.Status, resp.StatusCode)
	log.Printf("DEBUG: Response Headers: %v", resp.Header)
	log.Printf("DEBUG: Response Body: %s", string(respBodyBytes))

	// Check if response indicates success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("Customer.io Track API returned non-success status for email %s: %s. Body: %s", email, resp.Status, string(respBodyBytes))
		log.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}

	log.Printf("SUCCESS: Track API unsubscribe completed for email %s (status %s)", email, resp.Status)
	log.Printf("IMPORTANT: Customer should now be unsubscribed in Customer.io dashboard")

	return nil
}

// updateCustomerPausedAttribute updates the 'paused' attribute via Customer.io Track API.
func updateCustomerPausedAttribute(userID string) error {
	endpointURL := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", userID)

	// Track API uses a simple JSON payload with attributes
	payload := map[string]interface{}{
		"paused": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal Track API payload for UserID %s: %v", userID, err)
		return fmt.Errorf("error marshalling Track API payload: %w", err)
	}

	log.Printf("DEBUG: Attempting to update customer %s via PUT to %s", userID, endpointURL)
	log.Printf("DEBUG: Request payload: %s", string(payloadBytes))
	log.Printf("DEBUG: Using Site ID: %s, API Key: %s... (first 10 chars)", customerIOSiteID, customerIOAPIKey[:10])

	req, err := http.NewRequest(http.MethodPut, endpointURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("ERROR: Failed to create Track API request for UserID %s: %v", userID, err)
		return fmt.Errorf("error creating Track API request: %w", err)
	}

	// Track API uses Basic Auth: Site ID as username, API Key as password
	req.SetBasicAuth(customerIOSiteID, customerIOAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CustomerIO-Pauser/1.0")

	log.Printf("DEBUG: Request headers set - Content-Type: application/json, Authorization: Basic [REDACTED]")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send Track API request for UserID %s: %v", userID, err)
		return fmt.Errorf("error sending Track API request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Failed to read Track API response body for UserID %s: %v", userID, readErr)
		// Continue, but log this error.
	}

	log.Printf("DEBUG: Customer.io Track API response for UserID %s", userID)
	log.Printf("DEBUG: Response Status: %s (%d)", resp.Status, resp.StatusCode)
	log.Printf("DEBUG: Response Headers: %v", resp.Header)
	log.Printf("DEBUG: Response Body: %s", string(respBodyBytes))

	// Check if response indicates success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("Customer.io Track API returned non-success status for UserID %s: %s. Body: %s", userID, resp.Status, string(respBodyBytes))
		log.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}

	log.Printf("SUCCESS: Track API request completed for UserID %s (status %s)", userID, resp.Status)
	log.Printf("IMPORTANT: Customer attribute 'paused' should now be visible in Customer.io dashboard")
	log.Printf("  - Using Track API endpoint: %s", endpointURL)
	log.Printf("  - This API directly updates customer profiles in your Customer.io workspace")
	log.Printf("  - If attribute is still not visible, check Customer.io dashboard after 1-2 minutes")

	return nil
}

// basicAuthMiddleware provides HTTP Basic Authentication for protected routes
func basicAuthMiddleware(username, password string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Authorization header
		auth := c.Get("Authorization")
		if auth == "" {
			// No authorization header, request authentication
			c.Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			return c.Status(401).SendString("Unauthorized")
		}

		// Check if it's Basic auth
		if !strings.HasPrefix(auth, "Basic ") {
			c.Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			return c.Status(401).SendString("Unauthorized")
		}

		// Decode the base64 credentials
		encoded := auth[6:] // Remove "Basic " prefix
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			return c.Status(401).SendString("Unauthorized")
		}

		// Split username:password
		credentials := string(decoded)
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			c.Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			return c.Status(401).SendString("Unauthorized")
		}

		// Check credentials
		if parts[0] != username || parts[1] != password {
			c.Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			return c.Status(401).SendString("Unauthorized")
		}

		// Authentication successful, continue to next handler
		return c.Next()
	}
}

// handleResults handles the /results route with authentication and data visualization
func handleResults(c *fiber.Ctx) error {
	log.Printf("GET /results request received from IP: %s", c.IP())

	// Get summary data
	summary, err := getActionSummary()
	if err != nil {
		log.Printf("ERROR: Failed to get action summary: %v", err)
		return c.Status(500).SendString("Internal Server Error: Failed to retrieve summary data")
	}

	// Ensure all action types are present in summary (default to 0 if not found)
	if summary == nil {
		summary = make(map[string]int)
	}
	if _, exists := summary["PAUSE"]; !exists {
		summary["PAUSE"] = 0
	}
	if _, exists := summary["BBAU"]; !exists {
		summary["BBAU"] = 0
	}
	if _, exists := summary["UNSUBSCRIBE"]; !exists {
		summary["UNSUBSCRIBE"] = 0
	}

	// Get all records for display
	records, err := getAllRecordsForDisplay()
	if err != nil {
		log.Printf("ERROR: Failed to get records for display: %v", err)
		return c.Status(500).SendString("Internal Server Error: Failed to retrieve records")
	}

	log.Printf("Successfully retrieved %d records and summary data for /results", len(records))

	// Render the results template
	return c.Render("results", fiber.Map{
		"Summary": summary,
		"Records": records,
	})
}

// handleCSVDownload handles CSV download for specific action types
func handleCSVDownload(c *fiber.Ctx) error {
	action := c.Params("action")
	log.Printf("CSV download request for action: %s from IP: %s", action, c.IP())

	// Validate action type
	validActions := map[string]bool{
		"PAUSE":       true,
		"BBAU":        true,
		"UNSUBSCRIBE": true,
	}

	if !validActions[action] {
		log.Printf("ERROR: Invalid action type for CSV download: %s", action)
		return c.Status(400).SendString("Invalid action type")
	}

	// Get records for the specific action
	records, err := getRecordsByAction(action)
	if err != nil {
		log.Printf("ERROR: Failed to get records for action %s: %v", action, err)
		return c.Status(500).SendString("Internal Server Error: Failed to retrieve records")
	}

	// Create CSV content
	var csvBuffer bytes.Buffer
	writer := csv.NewWriter(&csvBuffer)

	// Write CSV header
	header := []string{"Date", "Email", "Action"}
	if err := writer.Write(header); err != nil {
		log.Printf("ERROR: Failed to write CSV header: %v", err)
		return c.Status(500).SendString("Internal Server Error: Failed to generate CSV")
	}

	// Write CSV rows
	for _, record := range records {
		row := []string{record.FormattedDate, record.Email, record.Action}
		if err := writer.Write(row); err != nil {
			log.Printf("ERROR: Failed to write CSV row: %v", err)
			return c.Status(500).SendString("Internal Server Error: Failed to generate CSV")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("ERROR: CSV writer error: %v", err)
		return c.Status(500).SendString("Internal Server Error: Failed to generate CSV")
	}

	// Set response headers for file download
	filename := fmt.Sprintf("%s_records_%s.csv", strings.ToLower(action), time.Now().Format("2006-01-02"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	log.Printf("Successfully generated CSV for action %s with %d records", action, len(records))
	return c.Send(csvBuffer.Bytes())
}

// handleClearRecords handles clearing all records from the database
func handleClearRecords(c *fiber.Ctx) error {
	log.Printf("Clear records request received from IP: %s", c.IP())

	// Clear all records
	err := clearAllRecords()
	if err != nil {
		log.Printf("ERROR: Failed to clear records: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Failed to clear records",
		})
	}

	log.Printf("Successfully cleared all records from database")
	return c.JSON(fiber.Map{
		"success": true,
		"message": "All records cleared successfully",
	})
}

// SubscriptionUpdate represents the subscription update request
type SubscriptionUpdate struct {
	Email         string            `json:"email"`
	Action        string            `json:"action"`
	Subscriptions map[string]string `json:"subscriptions"`
}

// handleUpdateSubscriptions handles updating individual brand subscriptions
func handleUpdateSubscriptions(c *fiber.Ctx) error {
	var req SubscriptionUpdate
	if err := c.BodyParser(&req); err != nil {
		log.Printf("ERROR: Failed to parse request body: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format",
		})
	}

	log.Printf("Updating subscriptions for email: %s", req.Email)

	// Update Customer.io attributes for each subscription
	err := updateCustomerSubscriptionAttributes(req.Email, req.Subscriptions)
	if err != nil {
		log.Printf("ERROR: Failed to update subscriptions for %s: %v", req.Email, err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update subscriptions",
		})
	}

	// Log to database
	if dbErr := insertEmailProcessingRecord(req.Email, "subscription_update"); dbErr != nil {
		log.Printf("WARNING: Failed to log subscription update to database for email %s: %v", req.Email, dbErr)
	}

	log.Printf("Successfully updated subscriptions for %s", req.Email)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Subscriptions updated successfully",
	})
}

// handleUnsubscribeAll handles unsubscribing from all brands
func handleUnsubscribeAll(c *fiber.Ctx) error {
	var req struct {
		Email  string `json:"email"`
		Action string `json:"action"`
	}
	if err := c.BodyParser(&req); err != nil {
		log.Printf("ERROR: Failed to parse request body: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format",
		})
	}

	log.Printf("Unsubscribing all for email: %s", req.Email)

	// Remove all subscription attributes and set unsubscribed to true
	err := unsubscribeAllBrands(req.Email)
	if err != nil {
		log.Printf("ERROR: Failed to unsubscribe all for %s: %v", req.Email, err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Failed to unsubscribe",
		})
	}

	// Log to database
	if dbErr := insertEmailProcessingRecord(req.Email, "unsubscribe_all"); dbErr != nil {
		log.Printf("WARNING: Failed to log unsubscribe all to database for email %s: %v", req.Email, dbErr)
	}

	log.Printf("Successfully unsubscribed all for %s", req.Email)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Unsubscribed from all brands successfully",
	})
}

// updateCustomerSubscriptionAttributes updates the subscription attributes for a customer
func updateCustomerSubscriptionAttributes(email string, subscriptions map[string]string) error {
	log.Printf("Updating subscription attributes for email: %s", email)

	// Build attributes map
	attributes := make(map[string]interface{})
	
	// Set each subscription attribute
	for key, value := range subscriptions {
		if value == "true" {
			attributes[key] = true
		} else if value == "false" {
			attributes[key] = false
		} else {
			// For "none" values, we don't set the attribute (it will be removed if it exists)
			attributes[key] = nil
		}
	}

	// Remove unsubscribed attribute if any subscriptions are active
	hasActiveSubscription := false
	for _, value := range subscriptions {
		if value == "true" {
			hasActiveSubscription = true
			break
		}
	}
	if hasActiveSubscription {
		attributes["unsubscribed"] = false
	}

	// Prepare the request payload
	requestBody := map[string]interface{}{
		"email":      email,
		"attributes": attributes,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("ERROR: Failed to marshal request body: %v", err)
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("ERROR: Failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerIOSiteID, customerIOAPIKey)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: HTTP request failed: %v", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("ERROR: Customer.io API returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully updated subscription attributes for %s", email)
	return nil
}

// unsubscribeAllBrands removes all subscription attributes and sets unsubscribed to true
func unsubscribeAllBrands(email string) error {
	log.Printf("Unsubscribing all brands for email: %s", email)

	// Build attributes map - set all subscriptions to null and unsubscribed to true
	attributes := map[string]interface{}{
		"unsubscribed": true,
		"sub_bbau":     nil,
		"sub_bbus":     nil,
		"sub_csau":     nil,
		"sub_csus":     nil,
		"sub_ffau":     nil,
		"sub_ffus":     nil,
		"sub_sbau":     nil,
		"sub_ppau":     nil,
	}

	// Prepare the request payload
	requestBody := map[string]interface{}{
		"email":      email,
		"attributes": attributes,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("ERROR: Failed to marshal request body: %v", err)
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://track.customer.io/api/v1/customers/%s", email)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("ERROR: Failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerIOSiteID, customerIOAPIKey)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: HTTP request failed: %v", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("ERROR: Customer.io API returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully unsubscribed all brands for %s", email)
	return nil
}
