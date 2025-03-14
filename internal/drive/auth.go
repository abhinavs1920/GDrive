package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

// getClient retrieves a Token from a local server.
func getClient(config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}

	// Read the token from the cache file.
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = getTokenFromWeb(config)

		// Save the token to the cache file.
		if err := saveToken(cacheFile, token); err != nil {
			log.Fatalf("Unable to save token to cache file. %v", err)
		}
	}

	return config.Client(context.Background(), token)
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

// getTokenFromWeb uses Config to request a Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
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
