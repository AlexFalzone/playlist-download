package youtube

import (
	"context"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"net/http"
	"os"
	"strings"
)

var httpClient = &http.Client{}
var durationMatchThreshold = 5

type SearchResult struct {
	Title     string
	Uploader  string
	URL       string
	Duration  string
	ID        string
	Live      bool
	Source    string
	ExtraInfo []string
}

// FindClosestMatchingVideo returns the best-match YouTube video ID for a given query.
func FindClosestMatchingVideo(searchQuery string, durationSeconds int) (string, error) {
	results, err := searchYouTubeAPI(searchQuery, 10)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no songs found for %s", searchQuery)
	}

	durationsMap, err := fetchDurationsForVideos(results) // vedi sotto
	if err == nil {
		for _, result := range results {
			dur := durationsMap[result.ID]
			if dur > 0 {
				allowedStart := durationSeconds - durationMatchThreshold
				allowedEnd := durationSeconds + durationMatchThreshold
				if dur >= allowedStart && dur <= allowedEnd {
					return result.ID, nil
				}
			}
		}
	}

	// Se non c’è corrispondenza di durata, fallback al primo risultato
	return results[0].ID, nil
}

func searchYouTubeAPI(query string, limit int64) ([]*SearchResult, error) {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing YOUTUBE_API_KEY environment variable")
	}

	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create youtube service: %w", err)
	}

	call := service.Search.List([]string{"id", "snippet"}).
		Q(query).
		Type("video").
		MaxResults(limit)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("youtube search error: %w", err)
	}

	if len(resp.Items) == 0 {
		return nil, nil
	}

	var results []*SearchResult
	for _, item := range resp.Items {
		if item.Id.Kind == "youtube#video" {
			vid := item.Id.VideoId
			title := item.Snippet.Title
			uploader := item.Snippet.ChannelTitle

			results = append(results, &SearchResult{
				Title:    title,
				Uploader: uploader,
				ID:       vid,
				URL:      "https://youtube.com/watch?v=" + vid,
				Source:   "youtube",
			})
		}
	}

	return results, nil
}

func fetchDurationsForVideos(results []*SearchResult) (map[string]int, error) {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	idStr := strings.Join(ids, ",")

	call := service.Videos.List([]string{"contentDetails"}).Id(idStr)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch videos info: %w", err)
	}

	durations := make(map[string]int)
	for _, item := range resp.Items {
		isoDur := item.ContentDetails.Duration
		seconds := parseISO8601Duration(isoDur)
		durations[item.Id] = seconds
	}
	return durations, nil
}

// parseISO8601Duration convert a duration string like “PT4M20S” into seconds.
func parseISO8601Duration(isoDur string) int {
	isoDur = strings.TrimPrefix(isoDur, "PT")

	var hours, mins, secs int
	if idx := strings.Index(isoDur, "H"); idx != -1 {
		_, err := fmt.Sscanf(isoDur[:idx], "%d", &hours)
		if err != nil {
			return 0
		}
		isoDur = isoDur[idx+1:]
	}
	if idx := strings.Index(isoDur, "M"); idx != -1 {
		_, err := fmt.Sscanf(isoDur[:idx], "%d", &mins)
		if err != nil {
			return 0
		}
		isoDur = isoDur[idx+1:]
	}
	if idx := strings.Index(isoDur, "S"); idx != -1 {
		_, err := fmt.Sscanf(isoDur[:idx], "%d", &secs)
		if err != nil {
			return 0
		}
		isoDur = isoDur[idx+1:]
	}

	return hours*3600 + mins*60 + secs
}

//
//func getContents(data []byte, index int) []byte {
//	container := fmt.Sprintf("[%d]", index)
//	contents, _, _, _ := jsonparser.Get(data, "contents", "twoColumnSearchResultsRenderer",
//		"primaryContents", "sectionListRenderer", "contents", container, "itemSectionRenderer", "contents")
//	return contents
//}
//
//// parseVideoDuration converts a duration string like "4:20" or "1:10:25" into seconds.
//func parseVideoDuration(durationStr string) int {
//	parts := strings.Split(durationStr, ":")
//	if len(parts) == 1 {
//		// only seconds
//		return toInt(parts[0])
//	} else if len(parts) == 2 {
//		// mm:ss
//		minutes := toInt(parts[0])
//		seconds := toInt(parts[1])
//		return (minutes * 60) + seconds
//	} else if len(parts) == 3 {
//		// hh:mm:ss
//		hours := toInt(parts[0])
//		minutes := toInt(parts[1])
//		seconds := toInt(parts[2])
//		return (hours * 3600) + (minutes * 60) + seconds
//	}
//	return 0
//}
//
//func toInt(s string) int {
//	var val int
//	_, err := fmt.Sscanf(s, "%d", &val)
//	if err != nil {
//		return 0
//	}
//	return val
//}
