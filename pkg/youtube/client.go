package youtube

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Client handles interactions with the YouTube API
type Client struct {
	service *youtube.Service
}

// VideoStat represents statistics for a YouTube video
type VideoStat struct {
	ID            string
	Title         string
	ViewCount     uint64
	LikeCount     uint64
	DislikeCount  uint64
	CommentCount  uint64
	PublishedAt   string
	EpisodeNumber int
}

// NewClient creates a new YouTube API client
func NewClient(credentialsFile string) (*Client, error) {
	ctx := context.Background()

	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}

	config, err := google.JWTConfigFromJSON(data, youtube.YoutubeReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	client := config.Client(ctx)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// ExtractPlaylistID extracts a playlist ID from a URL or returns the ID if already provided
func ExtractPlaylistID(playlistIDOrURL string) string {
	// Check if it's a URL
	if strings.Contains(playlistIDOrURL, "youtube.com") || strings.Contains(playlistIDOrURL, "youtu.be") {
		// Extract the list parameter from URL
		u, err := url.Parse(playlistIDOrURL)
		if err == nil {
			q := u.Query()
			if listID := q.Get("list"); listID != "" {
				return listID
			}
		}
	}
	
	// Return as is if it's already a playlist ID
	return playlistIDOrURL
}

// GetPlaylistVideos retrieves all video IDs from a playlist
func (c *Client) GetPlaylistVideos(playlistIDOrURL string) ([]string, error) {
	// Extract playlist ID from URL if needed
	playlistID := ExtractPlaylistID(playlistIDOrURL)
	
	var videoIDs []string
	nextPageToken := ""

	for {
		call := c.service.PlaylistItems.List([]string{"snippet", "contentDetails"}).
			PlaylistId(playlistID).
			MaxResults(50)

		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch playlist items: %w", err)
		}

		for _, item := range response.Items {
			videoIDs = append(videoIDs, item.ContentDetails.VideoId)
		}

		nextPageToken = response.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	return videoIDs, nil
}

// GetVideoStats retrieves statistics for a list of videos
func (c *Client) GetVideoStats(videoIDs []string) ([]VideoStat, error) {
	var stats []VideoStat
	
	// Process videos in batches of 50 (API limit)
	for i := 0; i < len(videoIDs); i += 50 {
		end := i + 50
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		
		batch := videoIDs[i:end]
		
		call := c.service.Videos.List([]string{"snippet", "statistics"}).Id(batch...)
		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch video statistics: %w", err)
		}

		for _, item := range response.Items {
			// YouTube API doesn't return dislike counts anymore, but we keep the field for consistency
			stats = append(stats, VideoStat{
				ID:           item.Id,
				Title:        item.Snippet.Title,
				ViewCount:    item.Statistics.ViewCount,
				LikeCount:    item.Statistics.LikeCount,
				DislikeCount: 0,
				CommentCount: item.Statistics.CommentCount,
				PublishedAt:  item.Snippet.PublishedAt,
			})
		}
	}

	// Sort videos by publication date (oldest first)
	SortVideosByPublishedDate(stats)
	
	// Add episode numbers (starting from 0 for the oldest video)
	for i := range stats {
		stats[i].EpisodeNumber = i
	}

	return stats, nil
}