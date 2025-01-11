package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"log"
	"os"
	"playlist-download/src/auth"
	"playlist-download/src/downloader"
	"playlist-download/src/parser"
	"playlist-download/src/utils"
	"strings"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()
	var outputDir string
	var workerCount int
	var cookies string

	rootCmd := &cobra.Command{
		Use: "playlist-download",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}

			spotifyURL := args[0]
			if spotifyURL == "" {
				fmt.Println("=> Spotify URL is required.")
				return cmd.Help()
			}

			finalDir, err := utils.EnsureDefaultOutputDir(outputDir)
			if err != nil {
				return err
			}
			outputDir = finalDir

			browserEnum, err := downloader.ParseBrowserCookieMode(cookies)
			if err != nil {
				return err
			}

			urlType, spotifyID, err := parser.ParseSpotifyURL(spotifyURL)
			if err != nil {
				return fmt.Errorf("error parsing URL: %w", err)
			}

			client, err := auth.InitSpotifyClient(ctx)
			if err != nil {
				return fmt.Errorf("authentication error: %w", err)
			}

			switch urlType {
			case parser.AlbumURL:
				if err := downloader.DownloadAlbum(ctx, client, spotifyID, outputDir, workerCount, browserEnum); err != nil {
					return err
				}
			case parser.PlaylistURL:
				if err := downloader.DownloadPlaylist(ctx, client, spotifyID, outputDir, workerCount, browserEnum); err != nil {
					return err
				}
			case parser.TrackURL:
				if err := downloader.DownloadTrack(ctx, client, spotifyID, outputDir, workerCount, browserEnum); err != nil {
					return err
				}
			default:
				fmt.Println("=> Only album, playlist, or track URLs are supported.")
				return cmd.Help()
			}

			return nil
		},
	}

	// Flag -o / --output
	rootCmd.Flags().StringVarP(
		&outputDir,
		"output",
		"o",
		"",
		"Specify the output directory (default is current directory)",
	)

	rootCmd.Flags().IntVarP(
		&workerCount,
		"workers",
		"w",
		3, // default
		"Number of concurrent workers (default is 3)",
	)

	rootCmd.Flags().StringVarP(
		&cookies,
		"cookies",
		"c",
		"",
		"Specify a browser where you are logged in to YouTube."+
			"it is used to take cookies"+
			"It is necessary for download age restricted content or similar (default is empty)"+
			"Currently supported browsers: Chrome, Firefox, Safari, Edge, Brave, Opera",
	)

	rootCmd.SetUsageTemplate(`
		Usage:
		  playlist-download [flags] [spotify_url]
		
		Examples:
		  playlist-download -c Brave -o "./music" -w 5 https://open.spotify.com/track/...
		  playlist-download --cookies Brave --output "./my_playlist" --workers 2 https://open.spotify.com/playlist/...
		  playlist-download https://open.spotify.com/album/...
		
		Flags:
		  -o, --output string    Specify the output directory (default is current directory)
		  -w, --workers int      Number of concurrent workers (default is 3)
          -c, --cookies string   Specify a browser where you are logged in to YouTube. It is used to take cookies. It is necessary for download age restricted content or similar (default is empty)
		  -h, --help             Help for this command
	`)

	if err := rootCmd.Execute(); err != nil {
		if strings.Contains(err.Error(), "quotaExceeded") {
			fmt.Println("Warning: YouTube quota exceeded. Some tracks not downloaded.")
			os.Exit(0)
		} else {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	}
}
