package parser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type SpotifyURLType int

const (
	UnknownURL SpotifyURLType = iota
	TrackURL
	AlbumURL
	PlaylistURL
)

// ParseSpotifyURL checks whether the provided string is a valid Spotify URL
// and extracts both the resource type and ID from it.
func ParseSpotifyURL(spotifyURL string) (SpotifyURLType, string, error) {
	parsed, err := url.Parse(spotifyURL)
	if err != nil || parsed.Host == "" {
		return UnknownURL, "", fmt.Errorf("invalid URL: %s", spotifyURL)
	}

	// Example of path: /track/XYZ or /playlist/XYZ
	splitPath := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(splitPath) < 2 {
		return UnknownURL, "", errors.New("URL path is too short")
	}

	resourceType := splitPath[0] // track, album, playlist
	spotifyID := splitPath[1]

	// Strip query parameters if present
	if idx := strings.Index(spotifyID, "?"); idx != -1 {
		spotifyID = spotifyID[:idx]
	}

	switch resourceType {
	case "track":
		return TrackURL, spotifyID, nil
	case "album":
		return AlbumURL, spotifyID, nil
	case "playlist":
		return PlaylistURL, spotifyID, nil
	default:
		return UnknownURL, "", fmt.Errorf("unsupported Spotify resource type: %s", resourceType)
	}
}
