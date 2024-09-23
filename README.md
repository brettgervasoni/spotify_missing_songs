# Spotify Missing Songs

Spotify Missing Songs is a Go program that helps you keep track of changes in your Spotify playlists, particularly focusing on identifying songs that have been removed without your knowledgefrom your playlists or Liked Songs.

## Features

- Scan your Liked Songs (Saved Tracks) for any missing tracks
- Scan other playlists for missing tracks
- Scan all playlists, including Liked Songs
- Scan only favorite playlists (as defined in a JSON file)
- Print all playlists
- Desktop notifications for missing songs (using beeep)
- Search functionality to find potential matches for missing songs

## Prerequisites

- Go 1.20 or later
- Spotify Developer account and API credentials

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/spotify-missing-songs.git
   cd spotify-missing-songs
   mkdir playlists
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Set up your Spotify API credentials:
   - Create a Spotify Developer account and create a new application at https://developer.spotify.com/dashboard/
   - In your application settings, set the redirect URI to `http://localhost:8081/callback`
   - Note your Client ID and Client Secret (you'll need these for environment variables, see below)

4. Set up environment variables:
   To run this program, you'll need to set two environment variables: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET. You have two options:

   Option 1: Set the environment variables directly:
   ```bash
   export SPOTIFY_CLIENT_ID="your_client_id"
   export SPOTIFY_CLIENT_SECRET="your_client_secret"
   ```

   Option 2 (macOS only): Use the keychain to store your credentials:
   
   First, add your credentials to the keychain:
   ```bash
   security add-generic-password -a "your_email@example.com" -s "Spotify API" "Client ID"
   security add-generic-password -a "your_email@example.com" -s "Spotify API" "Client Secret"
   ```

   Then, retrieve and set the values in your shell:
   ```bash
   export SPOTIFY_CLIENT_ID=$(security find-generic-password -a "your_email@example.com" -s "Spotify API" -w)
   export SPOTIFY_CLIENT_SECRET=$(security find-generic-password -a "your_email@example.com" -s "Spotify API Secret" -w)
   ```

   Replace "your_email@example.com" with your actual email address.

5. Run:
```bash
go run *.go --help
```

## Usage

Run the program with one of the following flags:

- `-o`: Scan other playlists excluding Liked Songs
- `-a`: Scan all playlists, including your Liked Songs (Saved Tracks)
- `-f`: Scan favorite playlists only (including Liked Songs), as defined by single comma-separated playlist IDs in a JSON file
- `-p`: Print all playlists
- `-h`: Show the help menu
