package tags

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/zmb3/spotify/v2"
)

// TagFileWithSpotifyMetadata applies metadata (artist, album, year, cover art) to an MP3 file.
func TagFileWithSpotifyMetadata(fileName string, trackData spotify.FullTrack, coverArt []byte) error {
	cleanTitle := removeUnsupportedRunes(trackData.Name)
	cleanArtist := removeUnsupportedRunes(joinArtists(trackData.Album.Artists))
	cleanAlbum := removeUnsupportedRunes(trackData.Album.Name)

	mp3File, err := id3v2.Open(fileName, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open mp3 file: %w", err)
	}
	defer func() {
		if closeErr := mp3File.Close(); closeErr != nil {
			log.Printf("error closing mp3 file: %v", closeErr)
		}
	}()

	// Impostiamo i campi base
	mp3File.SetTitle(cleanTitle)
	mp3File.SetArtist(cleanArtist)
	mp3File.SetAlbum(cleanAlbum)

	// Year
	year := extractYear(trackData.Album.ReleaseDate)
	mp3File.SetYear(strconv.Itoa(year))

	// Se abbiamo una coverArt condivisa (non nil), la usiamo
	if coverArt != nil && len(coverArt) > 0 {
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3v2.PTFrontCover,
			Description: "Front cover",
			Picture:     coverArt,
		}
		mp3File.AddAttachedPicture(pic)
	} else {
		// Nessuna copertina? Log e proseguiamo
		log.Printf("No album art provided for track: %s\n", trackData.Name)
	}

	if err = mp3File.Save(); err != nil {
		return fmt.Errorf("failed to save mp3 tag: %w", err)
	}
	return nil
}

func joinArtists(artists []spotify.SimpleArtist) string {
	var names []string
	for _, ar := range artists {
		names = append(names, ar.Name)
	}
	return strings.Join(names, ", ")
}

func extractYear(dateStr string) int {
	layout := "2006-01-02"
	if len(dateStr) == 4 {
		dateStr += "-01-01"
	}
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		return 0
	}
	return t.Year()
}

var invalidRunes = []rune{'⟁', '…', '’', '‘', '“', '”', '–', '—', '•', '♪', '♫', '♩', '♬', '♭', '♮'}

func removeUnsupportedRunes(s string) string {
	out := strings.Builder{}
	for _, r := range s {
		if isRuneInvalid(r) {
			out.WriteRune('-')
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func isRuneInvalid(r rune) bool {
	for _, x := range invalidRunes {
		if r == x {
			return true
		}
	}
	return false
}
