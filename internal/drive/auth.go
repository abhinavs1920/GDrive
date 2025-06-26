package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// AuthenticateGoogleDrive initializes Google Drive API client
func AuthenticateGoogleDrive() (*drive.Service, error) {
	ctx := context.Background()

	// Load credentials.json
	b, err := os.ReadFile("configs/credentials.json")
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// Parse credentials
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file: %v", err)
	}

	// Get token
	client := getClient(config)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Drive client: %v", err)
	}

	return service, nil
}

// getClient retrieves a Token from a local server and handles token refresh
func getClient(config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}

	// Read the token from the cache file.
	token, err := tokenFromFile(cacheFile)
	if err != nil || !token.Valid() {
		log.Println("Token not found or invalid, getting new token...")
		token = getTokenFromWeb(config)

		// Save the token to the cache file.
		if err := saveToken(cacheFile, token); err != nil {
			log.Fatalf("Unable to save token to cache file. %v", err)
		}
	}

	// Create a token source with auto-refresh
	tokenSource := config.TokenSource(context.Background(), token)

	return oauth2.NewClient(context.Background(), oauth2.ReuseTokenSource(token, tokenSource))
}

// tokenCacheFile returns the file path where the credentials are cached.
func tokenCacheFile() (string, error) {
	fname := "token.json"

	// Try to get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fname, err
	}

	return filepath.Join(homeDir, ".credentials", fname), nil
}

// getTokenFromWeb uses Config to request a Token with local server based authorization flow
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// Setup local server to handle the redirect
	config.RedirectURL = "http://localhost:8080/callback"
	
	// Create a channel to receive the authorization code
	codeChan := make(chan string)
	
	// Create a new ServeMux to avoid conflicts with the default one
	mux := http.NewServeMux()
	
	// Setup a local HTTP server to handle the callback
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	// Define the callback handler
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Error: No authorization code received")
			codeChan <- ""
			return
		}
		
		// Return a success page to the user
		fmt.Fprintf(w, "Authorization successful! You can close this window and return to the application.")
		
		// Send the code through the channel
		codeChan <- code
	})
	
	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	
	// Generate the authorization URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Please open the following URL in your browser to authorize the application:\n%v\n", authURL)
	
	// Wait for the code to be received
	code := <-codeChan
	
	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	if code == "" {
		log.Fatalf("No authorization code received")
	}
	
	// Exchange the authorization code for a token
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	
	return tok
}

// tokenFromFile retrieves a Token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open token file: %v", err)
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("failed to decode token: %v", err)
	}

	// Check if token is expired or about to expire soon
	if time.Until(tok.Expiry) < 5*time.Minute {
		return nil, fmt.Errorf("token expired or about to expire")
	}

	return tok, nil
}

// saveToken saves a Token to a file path.
func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)

	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return fmt.Errorf("Unable to create directory %v", err)
	}

	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}
