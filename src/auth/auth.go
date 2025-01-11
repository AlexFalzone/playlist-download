package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

// InitSpotifyClient initializes a Spotify client by reading the client ID and secret
// from environment variables: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET.
func InitSpotifyClient(ctx context.Context) (*spotify.Client, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, errors.New("SPOTIFY_CLIENT_ID or SPOTIFY_CLIENT_SECRET not set")
	}

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token: %w", err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	log.Println("=> Successfully authenticated with Spotify.")
	return client, nil
}
