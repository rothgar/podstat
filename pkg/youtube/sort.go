package youtube

import (
	"sort"
	"time"
)

// SortVideosByPublishedDate sorts videos from oldest to newest
func SortVideosByPublishedDate(stats []VideoStat) {
	sort.Slice(stats, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, stats[i].PublishedAt)
		timeJ, _ := time.Parse(time.RFC3339, stats[j].PublishedAt)
		return timeI.Before(timeJ)
	})
}