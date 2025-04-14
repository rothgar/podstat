# Justfile for podstat Docker commands
# https://github.com/casey/just

# Default recipe (shows help)
default:
    @just --list

# Build the Docker image
build:
    docker build -t podstat .

# Run the YouTube fetcher (with credentials)
youtube playlist credentials format="default" output="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "{{output}}" ]; then
        docker run --rm -v "{{credentials}}:/creds/youtube.json" \
            podstat fetch -p "{{playlist}}" -c "/creds/youtube.json" -f "{{format}}"
    else
        docker run --rm -v "{{credentials}}:/creds/youtube.json" \
            podstat fetch -p "{{playlist}}" -c "/creds/youtube.json" -f "{{format}}" > "{{output}}"
        echo "Results saved to {{output}}"
    fi

# Run with docker-compose
compose:
    docker-compose up

# Run shell inside container (for debugging)
shell:
    docker run --rm -it podstat /bin/sh