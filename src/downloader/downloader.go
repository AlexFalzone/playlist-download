package downloader

import (
	"context"
	"fmt"
	"github.com/zmb3/spotify/v2"
	"log"
	"path/filepath"
	"playlist-download/src/tags"
	"playlist-download/src/utils"
	yt "playlist-download/src/yt"
	"strings"
	"time"
)

type EnumCookies string

const (
	EnumCookiesNone EnumCookies = ""
	Chrome          EnumCookies = "chrome"
	Firefox         EnumCookies = "firefox"
	Edge            EnumCookies = "edge"
	Brave           EnumCookies = "brave"
	Safari          EnumCookies = "safari"
	Opera           EnumCookies = "opera"
)

func ParseBrowserCookieMode(input string) (EnumCookies, error) {
	input = strings.ToLower(input)
	switch input {
	case "", "none":
		return EnumCookiesNone, nil
	case "chrome":
		return Chrome, nil
	case "firefox":
		return Firefox, nil
	case "edge":
		return Edge, nil
	case "brave":
		return Brave, nil
	case "safari":
		return Safari, nil
	case "opera":
		return Opera, nil
	default:
		return EnumCookiesNone, fmt.Errorf("invalid cookies enum: '%s' (valid: chrome, firefox, edge, none)", input)
	}
}

func buildSearchQuery(track spotify.FullTrack) string {
	if len(track.Artists) == 0 {
		return utils.CleanTitleForSearch(track.Name)
	}

	mainArtist := track.Artists[0].Name
	title := utils.CleanTitleForSearch(track.Name)

	query := fmt.Sprintf("%s %s", mainArtist, title)

	return query
}

func sanitizeFileName(name string) string {
	name = utils.RemoveIllegalPathChars(name)
	return name
}

func downloadTrackWithRetry(videoID string, track spotify.FullTrack, outputDir string, maxRetries int, delay time.Duration, cookies EnumCookies) (string, error) {
	ytURL := "https://www.youtube.com/watch?v=" + videoID
	finalPath := filepath.Join(outputDir, sanitizeFileName(track.Name)+".mp3")

	cmdArgs := []string{
		"-f", "bestaudio",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--write-thumbnail",
		"--embed-thumbnail",
		"--embed-metadata",
		"-o", finalPath,

		ytURL,
	}

	if cookies != EnumCookiesNone {
		cmdArgs = append(cmdArgs, "--cookies-from-browser", string(cookies))
	}

	cmdArgs = append(cmdArgs, ytURL)

	output, err := utils.RunCmdWithRetry("yt-dlp", cmdArgs, maxRetries, delay)
	if err != nil {
		return "", fmt.Errorf("yt-dlp (retry) failed: %v\nCommand output: %s", err, string(output))
	}
	return finalPath, nil
}

func DownloadAlbum(ctx context.Context, client *spotify.Client, albumID string, outputDir string, workerCount int, cookies EnumCookies) error {
	album, err := client.GetAlbum(ctx, spotify.ID(albumID))
	if err != nil {
		return fmt.Errorf("failed to fetch album: %w", err)
	}
	if album == nil {
		return fmt.Errorf("album %s not found (empty response)", albumID)
	}

	var trackList []spotify.FullTrack
	for _, val := range album.Tracks.Tracks {
		trackList = append(trackList, spotify.FullTrack{
			SimpleTrack: val,
			Album:       album.SimpleAlbum,
		})
	}

	var coverArt []byte
	if len(album.Images) > 0 {
		coverArtURL := album.Images[0].URL
		coverArt, err = utils.DownloadFileWithRetry(coverArtURL, 3, 2*time.Second)
		if err != nil {
			log.Printf("Error downloading album art for album %s: %v", album.Name, err)
			coverArt = nil
		}
	}

	return DownloadTrackList(ctx, client, trackList, outputDir, coverArt, workerCount, cookies)
}

func DownloadPlaylist(ctx context.Context, client *spotify.Client, playlistID string, outputDir string, workerCount int, cookies EnumCookies) error {
	playlistTracks, err := client.GetPlaylistTracks(ctx, spotify.ID(playlistID))
	if err != nil {
		return fmt.Errorf("failed to fetch playlist: %w", err)
	}

	var trackList []spotify.FullTrack
	for _, t := range playlistTracks.Tracks {
		trackList = append(trackList, t.Track)
	}

	// Pagination
	for {
		err := client.NextPage(ctx, playlistTracks)
		if err != nil {
			if err.Error() == spotify.ErrNoMorePages.Error() {
				break
			}
			return fmt.Errorf("error paginating playlist: %w", err)
		}
		for _, t := range playlistTracks.Tracks {
			trackList = append(trackList, t.Track)
		}
	}

	playlist, err := client.GetPlaylist(ctx, spotify.ID(playlistID))
	if err != nil {
		log.Printf("Cannot fetch playlist details: %v", err)
	}
	var coverArt []byte
	if playlist != nil && len(playlist.Images) > 0 {
		coverArtURL := playlist.Images[0].URL
		coverArt, err = utils.DownloadFileWithRetry(coverArtURL, 3, 2*time.Second)
		if err != nil {
			log.Printf("Error downloading playlist cover art: %v", err)
			coverArt = nil
		}
	}

	return DownloadTrackList(ctx, client, trackList, outputDir, coverArt, workerCount, cookies)
}

func DownloadTrack(ctx context.Context, client *spotify.Client, trackID string, outputDir string, workerCount int, cookies EnumCookies) error {
	song, err := client.GetTrack(ctx, spotify.ID(trackID))
	if err != nil {
		return fmt.Errorf("failed to fetch track: %w", err)
	}

	ft := spotify.FullTrack{
		SimpleTrack: song.SimpleTrack,
		Album:       song.Album,
	}

	var coverArt []byte
	if len(song.Album.Images) > 0 {
		coverArtURL := song.Album.Images[0].URL
		coverArt, err = utils.DownloadFile(coverArtURL)
		if err != nil {
			log.Printf("Error downloading track cover art: %v", err)
			coverArt = nil
		}
	}

	return DownloadTrackList(ctx, client, []spotify.FullTrack{ft}, outputDir, coverArt, workerCount, cookies)
}

func DownloadTrackList(
	ctx context.Context,
	client *spotify.Client,
	tracks []spotify.FullTrack,
	outputDir string,
	sharedCoverArt []byte,
	workerCount int,
	cookies EnumCookies,
) error {
	fmt.Printf("Found %d tracks.\n", len(tracks))
	fmt.Println("Searching and downloading tracks with", workerCount, "workers...")

	// 1. Create the channels
	jobs := make(chan spotify.FullTrack, len(tracks))
	results := make(chan error, len(tracks))

	// 2. Start the workers
	for w := 0; w < workerCount; w++ {
		go workerFunc(ctx, jobs, results, outputDir, sharedCoverArt, cookies)
	}

	// 3. Send the tracks to the workers
	for _, track := range tracks {
		jobs <- track
	}
	close(jobs)

	// 4. Pick up the results from the workers
	var finalErr error
	for i := 0; i < len(tracks); i++ {
		err := <-results
		if err != nil {
			finalErr = err
		}
	}

	fmt.Println("Download complete!")
	return finalErr
}

func workerFunc(ctx context.Context, jobs <-chan spotify.FullTrack, results chan<- error, outputDir string, coverArt []byte, cookies EnumCookies) {
	for track := range jobs {
		err := processSingleTrack(ctx, track, outputDir, coverArt, cookies)
		results <- err
	}
}

func processSingleTrack(ctx context.Context, track spotify.FullTrack, outputDir string, coverArt []byte, cookies EnumCookies) error {
	query := buildSearchQuery(track)
	trackDurationSec := int(track.Duration) / 1000

	// 1. Find the YouTube video ID
	videoID, err := yt.FindClosestMatchingVideo(query, trackDurationSec)
	if err != nil {
		log.Printf("Error finding YouTube match for '%s': %v\n", track.Name, err)
		return err
	}

	// 2. Download the track as MP3 using retry
	fileName, err := downloadTrackWithRetry(videoID, track, outputDir, 3, 2*time.Second, cookies)
	if err != nil {
		log.Printf("Error downloading '%s': %v\n", track.Name, err)
		return err
	}

	// 3. Tag the downloaded MP3 file
	tagErr := tags.TagFileWithSpotifyMetadata(fileName, track, coverArt)
	if tagErr != nil {
		log.Printf("Error tagging '%s': %v\n", track.Name, tagErr)
		return tagErr
	}

	log.Printf("Successfully downloaded and tagged '%s'\n", track.Name)
	return nil
}
