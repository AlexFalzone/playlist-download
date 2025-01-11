# Playlist-Download

Playlist-Download is a CLI command that allows you to download songs from Spotify (album, playlist or single track) with
search and download from YouTube, automatically managing MP3 metadata (title, artist, album, cover, etc.).
Internally it relies on Spotify APIs for song extraction, YouTube APIs for video ID lookup and yt-dlp for the actual
download of audio in MP3 format.

PS. I don't know how much legal this is, but it works for now.

## Contents

- [How](#how)
- [Dependencies](#dependencies)
- [Installation](#installation)
- [Limits and cautions](#limits-and-cautions)

### How

- **Spotify extraction**:
  Given a URL of an album, playlist or single track on Spotify, the program gets the related metadata (song name,
  artist, cover).
  It uses the credentials defined in the SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables.

- **YouTube search**:
  Uses the YouTube Data API (via the YOUTUBE_API_KEY key) to find the most suitable video, also crossing the song
  duration for greater precision.
  If it doesn't find a match for the duration, it takes the first result.

- **Download with yt-dlp**:
  Download the audio in MP3 (embedded thumbnail and metadata support).
  If the videos are age-restricted, it is possible to specify the browser from which to copy the cookies (e.g. --cookies
  chrome), so that yt-dlp can authenticate itself.

- **Metadata management**:
  Once downloaded, update the MP3 file's ID3v2 tags (title, artist, album, year, cover art) using the
  github.com/bogem/id3v2 library.
  Unsupported special characters are replaced to avoid encoding errors.

  (**Note**: more fine-tuning is needed for bettere metadata management)

- **Parellelism**:
  With the *--workers* option you can define how many goroutines (workers) will process the songs in parallel, speeding
  up
  the download (but risking consuming the YouTube Data API quota more quickly).

### Dependencies

- Go
- yt-dlp
- ffmpeg
- An *.env* file with the following variables:
    - SPOTIFY_CLIENT_ID
    - SPOTIFY_CLIENT_SECRET
    - YOUTUBE_API_KEY
- Cookies (optional)

### Installation

- Clone the repository and enter the directory
  ```shell
    git clone 
  ```
  ```shell
    cd playlist-download
  ```
- Install the dependencies
- Compile
  ```shell
    go build
  ```

### Limits and cautions

- **YouTube Quota**:
  You have a daily limit on searches on YouTube Data API. If you download a lot of playlists in a short time, you may
  receive the quotaExceeded error.

- **Video Age-Restricted**:
  If the song on YouTube requires login (18+), you need to pass cookies with -c (e.g. -c chrome) to allow yt-dlp to
  authenticate.

- **Unsupported characters**:
  ID3v2.3 tags may give errors with some Unicode characters. A list of characters to replace has been implemented to
  avoid problems. If you encounter any issues, please open an issue or just fix it in the code.

- **Naming**:
  Any slashes, colons or other illegal characters in filenames are replaced with _ (RemoveIllegalPathChars method).