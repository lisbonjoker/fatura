CDN := https://cdn.jsdelivr.net/npm

.PHONY: fonts build

fonts:
	mkdir -p IBMPlex
	curl -fsSL -o IBMPlex/IBMPlexSans-Regular.ttf \
	  "$(CDN)/@ibm/plex-sans@1.1.0/fonts/complete/ttf/IBMPlexSans-Regular.ttf"
	curl -fsSL -o IBMPlex/IBMPlexSans-SemiBold.ttf \
	  "$(CDN)/@ibm/plex-sans@1.1.0/fonts/complete/ttf/IBMPlexSans-SemiBold.ttf"
	curl -fsSL -o IBMPlex/IBMPlexMono-Regular.ttf \
	  "$(CDN)/@ibm/plex-mono@1.1.0/fonts/complete/ttf/IBMPlexMono-Regular.ttf"
	curl -fsSL -o IBMPlex/IBMPlexMono-Medium.ttf \
	  "$(CDN)/@ibm/plex-mono@1.1.0/fonts/complete/ttf/IBMPlexMono-Medium.ttf"

build: fonts
	go build ./...
