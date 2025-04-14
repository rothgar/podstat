package cmd

import (
	"fmt"

	"github.com/rothgar/podstat/pkg/youtube"
	"github.com/spf13/cobra"
)

var (
	playlistID      string
	credentialsFile string
	outputFormat    string
	
	// DB endpoints
	influxEndpoint  string
	influxDB        string
	influxUser      string
	influxPassword  string
	victoriaEndpoint string
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch YouTube podcast statistics",
	Long:  `Fetch and display statistics for videos in a YouTube playlist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fetchYouTubeStats()
	},
}

func fetchYouTubeStats() error {
	// Check if we have YouTube data parameters
	if playlistID == "" || credentialsFile == "" {
		return fmt.Errorf("both playlist ID and credentials file are required")
	}

	// Get YouTube data
	var ytStats []youtube.VideoStat
	client, err := youtube.NewClient(credentialsFile)
	if err != nil {
		return fmt.Errorf("failed to create YouTube client: %w", err)
	}

	videos, err := client.GetPlaylistVideos(playlistID)
	if err != nil {
		return fmt.Errorf("failed to get playlist videos: %w", err)
	}

	ytStats, err = client.GetVideoStats(videos)
	if err != nil {
		return fmt.Errorf("failed to get video statistics: %w", err)
	}

	// Check if we should send to a database directly
	if influxEndpoint != "" {
		if influxDB == "" {
			return fmt.Errorf("database name is required when using InfluxDB endpoint")
		}
		
		err := youtube.SendToInfluxDB(ytStats, influxEndpoint, influxDB, influxUser, influxPassword)
		if err != nil {
			return fmt.Errorf("failed to send to InfluxDB: %w", err)
		}
		
		fmt.Printf("Successfully sent %d records to InfluxDB at %s\n", len(ytStats), influxEndpoint)
		return nil
	}
	
	if victoriaEndpoint != "" {
		err := youtube.SendToVictoriaMetrics(ytStats, victoriaEndpoint)
		if err != nil {
			return fmt.Errorf("failed to send to VictoriaMetrics: %w", err)
		}
		
		fmt.Printf("Successfully sent %d records to VictoriaMetrics at %s\n", len(ytStats), victoriaEndpoint)
		return nil
	}

	// Format based on output format flag
	switch outputFormat {
	case "influx":
		// InfluxDB line protocol format
		fmt.Print(youtube.FormatInfluxLineProtocol(ytStats))
	case "prometheus":
		// Prometheus exposition format
		fmt.Print(youtube.FormatPrometheus(ytStats))
	case "victoria":
		// VictoriaMetrics JSON format for import API
		fmt.Print(youtube.FormatVictoriaMetrics(ytStats))
	case "json":
		// JSON format
		fmt.Println(youtube.StatsToJSON(ytStats))
	default:
		// Default human-readable format
		for _, stat := range ytStats {
			fmt.Printf("Episode %d | %s | %s | Views: %d | Likes: %d | Comments: %d\n", 
				stat.EpisodeNumber, stat.Title, stat.ID, stat.ViewCount, stat.LikeCount, stat.CommentCount)
		}
	}

	return nil
}

func init() {
	// YouTube parameters
	fetchCmd.Flags().StringVarP(&playlistID, "playlist", "p", "", "YouTube playlist ID or URL")
	fetchCmd.Flags().StringVarP(&credentialsFile, "credentials", "c", "", "Path to the YouTube API credentials file")
	
	// Output format parameter
	fetchCmd.Flags().StringVarP(&outputFormat, "format", "f", "default", 
		"Output format (default, json, influx, prometheus, victoria)")
		
	// Database connection parameters
	fetchCmd.Flags().StringVar(&influxEndpoint, "influx-endpoint", "", 
		"InfluxDB endpoint URL (e.g., http://localhost:8086)")
	fetchCmd.Flags().StringVar(&influxDB, "influx-db", "", 
		"InfluxDB database name")
	fetchCmd.Flags().StringVar(&influxUser, "influx-user", "", 
		"InfluxDB username (optional)")
	fetchCmd.Flags().StringVar(&influxPassword, "influx-password", "", 
		"InfluxDB password (optional)")
	fetchCmd.Flags().StringVar(&victoriaEndpoint, "victoria-endpoint", "", 
		"VictoriaMetrics endpoint URL (e.g., http://localhost:8428)")
}