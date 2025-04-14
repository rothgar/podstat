package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GetCurrentTimestamp returns the current timestamp in nanoseconds
func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

// EscapeStringForInflux escapes a string to be safe for InfluxDB line protocol
func EscapeStringForInflux(s string) string {
	s = strings.ReplaceAll(s, " ", "\\ ")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "=", "\\=")
	return s
}

// EscapeStringForPrometheus escapes a string to be safe for Prometheus exposition format
func EscapeStringForPrometheus(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}

// StatsToJSON converts the stats to JSON format
func StatsToJSON(stats []VideoStat) string {
	type TimeSeriesPoint struct {
		VideoID      string `json:"video_id"`
		Title        string `json:"title"`
		ViewCount    uint64 `json:"view_count"`
		LikeCount    uint64 `json:"like_count"`
		CommentCount uint64 `json:"comment_count"`
		PublishedAt  string `json:"published_at"`
		Timestamp    string `json:"timestamp"`
		Episode      int    `json:"episode"`
	}

	now := time.Now().Format(time.RFC3339)
	var points []TimeSeriesPoint

	for _, stat := range stats {
		points = append(points, TimeSeriesPoint{
			VideoID:      stat.ID,
			Title:        stat.Title,
			ViewCount:    stat.ViewCount,
			LikeCount:    stat.LikeCount,
			CommentCount: stat.CommentCount,
			PublishedAt:  stat.PublishedAt,
			Timestamp:    now,
			Episode:      stat.EpisodeNumber,
		})
	}

	jsonData, err := json.MarshalIndent(points, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error converting to JSON: %v", err)
	}

	return string(jsonData)
}

// FormatInfluxLineProtocol formats stats as InfluxDB line protocol
func FormatInfluxLineProtocol(stats []VideoStat) string {
	var buf strings.Builder
	timestamp := GetCurrentTimestamp()
	for _, stat := range stats {
		safeTitle := EscapeStringForInflux(stat.Title)
		line := fmt.Sprintf("youtube_stats,video_id=%s,title=%s,episode=%d views=%d,likes=%d,dislikes=%d,comments=%d %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.ViewCount, stat.LikeCount, stat.DislikeCount, stat.CommentCount, timestamp)
		buf.WriteString(line)
	}
	return buf.String()
}

// FormatVictoriaMetrics formats stats for VictoriaMetrics import API
func FormatVictoriaMetrics(stats []VideoStat) string {
	var buf strings.Builder
	timestamp := time.Now().Unix() * 1000 // Convert to milliseconds for VictoriaMetrics

	// Format each stat as a separate metric in prometheus format
	for _, stat := range stats {
		safeTitle := EscapeStringForPrometheus(stat.Title)
		
		// Add views metric
		buf.WriteString(fmt.Sprintf("youtube_video_views{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.ViewCount, timestamp))
		
		// Add likes metric
		buf.WriteString(fmt.Sprintf("youtube_video_likes{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.LikeCount, timestamp))
		
		// Add dislikes metric  
		buf.WriteString(fmt.Sprintf("youtube_video_dislikes{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.DislikeCount, timestamp))
		
		// Add comments metric
		buf.WriteString(fmt.Sprintf("youtube_video_comments{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.CommentCount, timestamp))
	}
	
	return buf.String()
}

// FormatPrometheus formats stats for Prometheus exposition format
func FormatPrometheus(stats []VideoStat) string {
	var buf strings.Builder
	
	buf.WriteString("# HELP youtube_video_views Number of views for YouTube video\n")
	buf.WriteString("# TYPE youtube_video_views gauge\n")
	
	for _, stat := range stats {
		safeTitle := EscapeStringForPrometheus(stat.Title)
		
		viewLine := fmt.Sprintf("youtube_video_views{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.ViewCount)
		likesLine := fmt.Sprintf("youtube_video_likes{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.LikeCount)
		dislikesLine := fmt.Sprintf("youtube_video_dislikes{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.DislikeCount)
		commentsLine := fmt.Sprintf("youtube_video_comments{video_id=\"%s\",title=\"%s\",episode=\"%d\"} %d\n", 
			stat.ID, safeTitle, stat.EpisodeNumber, stat.CommentCount)
		
		buf.WriteString(viewLine)
		buf.WriteString(likesLine)
		buf.WriteString(dislikesLine)
		buf.WriteString(commentsLine)
	}
	
	return buf.String()
}

// SendToInfluxDB sends stats directly to an InfluxDB instance
func SendToInfluxDB(stats []VideoStat, endpoint, database, username, password string) error {
	data := FormatInfluxLineProtocol(stats)
	
	// Build the URL with query parameters for database
	url := endpoint
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "write?db=" + database
	
	if username != "" && password != "" {
		url += "&u=" + username + "&p=" + password
	}
	
	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check for errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("InfluxDB request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// SendToVictoriaMetrics sends stats directly to a VictoriaMetrics instance
func SendToVictoriaMetrics(stats []VideoStat, endpoint string) error {
	data := FormatVictoriaMetrics(stats)
	
	// Make sure endpoint is properly formatted
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	
	// Use the proper endpoint for plaintext metrics
	url := endpoint + "api/v1/import/prometheus/metrics"
	
	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Set Content-Type to Prometheus text format
	req.Header.Set("Content-Type", "text/plain")
	
	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check for errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("VictoriaMetrics request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}