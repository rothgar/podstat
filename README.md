# Podstat

A CLI tool to fetch and display podcast statistics from YouTube and Transistor.fm, with support for time series database formats.

## Installation

```
go get github.com/rothgar/podstat
```

## Setup

### YouTube Setup

1. Create a Google Cloud project at https://console.cloud.google.com/
2. Enable the YouTube Data API v3
3. Create a service account and download credentials JSON file

## Usage

```
# Fetch statistics from YouTube
podstat fetch --source youtube -p PLAYLIST_ID -c /path/to/credentials.json [-f FORMAT]

### Examples

```
# Using direct YouTube playlist ID
podstat fetch -p "PLlrxD0HtieHi0MzaW0n7p4v_2MpN7LHj7" -c "./credentials.json"

# Using YouTube playlist URL
podstat fetch -p "https://www.youtube.com/playlist?list=PLlrxD0HtieHi0MzaW0n7p4v_2MpN7LHj7" -c "./credentials.json"

# Output to different formats
podstat fetch -p "PLlrxD0HtieHi0MzaW0n7p4v_2MpN7LHj7" -c "./credentials.json" -f influx > youtube_stats.txt
```
## Arguments

### YouTube Arguments
- `-p, --playlist`: YouTube playlist ID or URL
- `-c, --credentials`: Path to the YouTube API credentials file

### General Arguments
- `-f, --format`: Output format (default, json, influx, prometheus, victoria)

### Database Connection Arguments
- `--influx-endpoint`: InfluxDB endpoint URL (e.g., http://localhost:8086)
- `--influx-db`: InfluxDB database name
- `--influx-user`: InfluxDB username (optional)
- `--influx-password`: InfluxDB password (optional)
- `--victoria-endpoint`: VictoriaMetrics endpoint URL (e.g., http://localhost:8428)

## Episode Numbering

The tool automatically:
1. Retrieves all episodes from the source
2. Sorts them by publication date (oldest first) 
3. Assigns episode numbers starting from 0

## Time Series Database Integration

### InfluxDB

You can send data to InfluxDB in two ways:

#### 1. Direct Push to InfluxDB

Push metrics directly to an InfluxDB instance:

```
# For YouTube
podstat fetch -p PLAYLIST_ID -c credentials.json --influx-endpoint http://localhost:8086 --influx-db podcast_metrics

# With authentication
podstat fetch -p PLAYLIST_ID -c credentials.json --influx-endpoint http://localhost:8086 --influx-db podcast_metrics --influx-user username --influx-password password
```

#### 2. Pipe to InfluxDB CLI

Output in InfluxDB line protocol format and pipe to the InfluxDB CLI:

```
# For YouTube
podstat fetch -p PLAYLIST_ID -c credentials.json -f influx | influx write -b podcast_metrics

# For Transistor.fm
podstat fetch -s SHOW_ID -k API_KEY -f influx | influx write -b podcast_metrics
```

### Prometheus

Create a script that outputs to a file that Prometheus can scrape:

```bash
#!/bin/bash
# YouTube stats
podstat fetch -p PLAYLIST_ID -c credentials.json -f prometheus > /path/to/prometheus/textfile_collector/podcast.prom

# Transistor.fm stats
podstat fetch -s SHOW_ID -k API_KEY -f prometheus >> /path/to/prometheus/textfile_collector/podcast.prom
```

### VictoriaMetrics

You can send data to VictoriaMetrics in two ways:

#### 1. Direct Push to VictoriaMetrics

Push metrics directly to a VictoriaMetrics instance:

```bash
# YouTube stats
podstat fetch -p PLAYLIST_ID -c credentials.json --victoria-endpoint http://localhost:8428
```

#### 2. Manual Import via HTTP API

Output metrics in Prometheus-compatible format and send them via curl:

```bash
# YouTube stats
podstat fetch -p PLAYLIST_ID -c credentials.json -f victoria | curl -d @- -H "Content-Type: text/plain" -X POST 'http://victoriametrics:8428/api/v1/import/prometheus/metrics'

# Or save to file for import
podstat fetch -p PLAYLIST_ID -c credentials.json -f victoria > metrics.txt
curl -d @metrics.txt -H "Content-Type: text/plain" -X POST 'http://victoriametrics:8428/api/v1/import/prometheus/metrics'
```

### Graphite/Grafana

Use the JSON format and import via HTTP API or use InfluxDB as a data source.

## Docker Support

This tool includes Docker support for easy deployment and consistent environment.

### Basic Docker Usage

```bash
# Build the Docker image
docker build -t podstat .

# Run with the Docker image for YouTube
docker run --rm -v /path/to/credentials.json:/creds.json podstat fetch -p "your_playlist_id" -c "/creds.json"

# Run with the Docker image for Transistor.fm
docker run --rm podstat fetch -s "your_show_id" -k "your_api_key"
```

### Full Stack with Docker Compose

The included `docker-compose.yml` sets up a complete monitoring stack with:

- Podstat application (automatically pushes metrics to VictoriaMetrics)
- VictoriaMetrics (time series database)
- Grafana (visualization and dashboarding)

#### Key Features

- All services are connected via a dedicated Docker network
- VictoriaMetrics is automatically configured as a Grafana datasource
- Podstat is configured to push metrics directly to VictoriaMetrics
- All container data is persisted in Docker volumes

#### Getting Started

To use this setup:

1. Place your YouTube API credentials in a file named `credentials.json` in the project root
2. Edit the `docker-compose.yml` file to set your correct playlist ID
3. Start all services with:

```bash
docker-compose up -d
```

4. Access Grafana at http://localhost:3000 (username: admin, password: admin)
   - A pre-configured datasource for VictoriaMetrics is automatically set up
   - A dashboard for YouTube statistics is automatically loaded

5. Schedule regular metric collection (optional):
   To collect metrics regularly, you can add a cron job or use the Docker restart policy.

```bash
# Example to run every hour
docker-compose restart podstat

# Or create a simple script with a cron job
# 0 * * * * cd /path/to/podstat && docker-compose restart podstat
```

