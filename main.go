package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gen2brain/beeep"
	"github.com/zmb3/spotify"
)

var auth = spotify.NewAuthenticator("http://localhost:8081/callback", spotify.ScopeUserLibraryRead, spotify.ScopePlaylistReadPrivate)
var client spotify.Client

const (
	tokenFile                 = ".token.json"
	playlistDirectory         = "playlists"
	savedSongsBaseFilename    = "savedSongs"
	favouritePlaylistFilename = "favourite-playlists.json"
)

var (
	otherPlaylists = flag.Bool("o", false, "Scan other playlists excluding Liked Songs")
	allPlaylists   = flag.Bool("a", false, "Scan all playlists, including your Liked Songs (Saved Tracks)")
	favouritesOnly = flag.Bool("f", false, "Favourite playlists only (including Liked Songs), as defined by single comma seperated playlist IDs in a json file")
	printPlaylists = flag.Bool("p", false, "Print all playlists")
	help           = flag.Bool("h", false, "Show this help menu")
)

var favouritePlaylists map[string]string

func main() {
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		fmt.Println("\n\nDefaults to scanning Liked Songs (Saved Tracks) only.")
		return
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("Authentication error: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables must be set")
	}

	auth.SetAuthInfo(clientID, clientSecret) // set Spotify API credentials

	if fileExists(tokenFile) {
		// Read the token from the file
		savedToken, err := readTokenFromFile()
		if err != nil {
			log.Fatal(err)
		}

		// Use the savedToken as needed
		client = auth.NewClient(savedToken)
	} else {
		fmt.Println("Token file does not exist. Authentication required.")

		// Ask the user to log in to their Spotify account and use a callback function to capture the response
		http.HandleFunc("/callback", completeAuth)
		go http.ListenAndServe("localhost:8081", nil)

		url := auth.AuthURL("state")
		fmt.Println("Please log in to Spotify by visiting the following page:", url)
		fmt.Println("Press Enter after authenticating") // could just use a channel to automatically signal when its ready

		// press enter to continue
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
	}

	if *printPlaylists {
		playlists, err := getAllPlaylists(client)
		if err != nil {
			log.Fatal("Failed to get all playlists:", err)
		}
		for _, playlist := range playlists {
			fmt.Printf("ID: %s, Name: %s\n", playlist.ID, playlist.Name)
		}
		return
	}

	if *favouritesOnly && fileExists(favouritePlaylistFilename) {
		var err error
		favouritePlaylists, err = readJsonFromFile(favouritePlaylistFilename)
		if err != nil {
			log.Fatalln(err)
		}
	}

	doActions()
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token("state", r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}

	// Save the token to a file
	err = saveTokenToFile(tok)
	if err != nil {
		log.Fatal(err)
	}

	// Create a Spotify client using the obtained access token
	client = auth.NewClient(tok)
}

func doActions() {
	if !*otherPlaylists || *allPlaylists || *favouritesOnly {
		fmt.Println("Fetching Saved Tracks...")
		savedTracks, err := getSavedTracks(client)
		if err != nil {
			log.Fatal(err)
		}

		// Process and print the user's Saved Tracks
		savedSongs := make(map[string]string)
		for _, track := range savedTracks {
			savedSongs[string(track.ID)] = fmt.Sprintf("%s by %s", track.Name, getArtistsNames(track.Artists))
		}

		savedSongsFilename := path.Join(playlistDirectory, savedSongsBaseFilename+".json")
		if fileExists(savedSongsFilename) {
			previouslySavedSongs, err := readJsonFromFile(savedSongsFilename)
			if err != nil {
				log.Fatalln("Failed to read from file:", err)
			}

			missingSongs, searchResults := findMissingSongs(previouslySavedSongs, savedSongs)
			if len(missingSongs) > 0 {
				message := "The following songs are missing from your Saved Tracks list:"
				fmt.Println(message)

				var missingSongsList strings.Builder
				for id, song := range missingSongs {
					songMessage := fmt.Sprintf("\t- ID: %s, Song: %s\n", id, song)
					fmt.Print(songMessage)
					missingSongsList.WriteString(songMessage)
				}

				notificationMessage := fmt.Sprintf("%s\n%s", message, missingSongsList.String())
				beeep.Notify("Spotify Playlist Manager", notificationMessage, "")
				beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)

				if len(searchResults) > 0 {
					fmt.Println("\nSearch results for missing songs:")
					for id, results := range searchResults {
						fmt.Printf("For missing song (ID: %s): %s\n", id, missingSongs[id])
						for trackID, trackInfo := range results {
							fmt.Printf("\t- ID: %s, Song: %s\n", trackID, trackInfo)
						}
					}
				}
			} else {
				message := "No songs are missing from your Saved Tracks list - yay"
				fmt.Println(message)
				beeep.Notify("Spotify Playlist Manager", message, "")
				beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
			}
		}

		writeJsonToFile(savedSongsFilename, savedSongs)
		fmt.Println("Updated song cache with the latest list")
	}

	// Process other playlists
	if *otherPlaylists || *allPlaylists || *favouritesOnly {
		// get all playlists
		playlists, err := getAllPlaylists(client)
		if err != nil {
			log.Fatal(err)
		}

		if *favouritesOnly {
			fmt.Printf("\nSearching for favourite playlists\n\n")
		}

		for _, playlist := range playlists {
			if *favouritesOnly {
				if _, ok := favouritePlaylists[string(playlist.ID)]; !ok {
					continue
				}
			}

			// exclude Discover Weekly
			if playlist.Name == "Discover Weekly" || playlist.Name == "Listen Later" {
				break
			}

			// exclude Radio created playlists
			radioString := "Radio"
			if len(playlist.Name) > len(radioString) {
				if playlist.Name[len(playlist.Name)-len(radioString):] != radioString {
					fmt.Printf("ID: %s, Name: %s\n", playlist.ID, playlist.Name)
					processPlaylist(client, playlist)
				}
			} else {
				fmt.Printf("ID: %s, Name: %s\n", playlist.ID, playlist.Name)
				processPlaylist(client, playlist)
			}
		}
	}
}

func processPlaylist(client spotify.Client, p spotify.SimplePlaylist) {
	fmt.Println("Processing Playlist:", p.Name)

	savedSongs := make(map[string]string)
	offset := 0
	limit := 100 // grabbing 100 tracks at a time

	// loop through all the tracks in the playlist using pagination
	for {
		tracksPage, err := client.GetPlaylistTracksOpt(p.ID, &spotify.Options{
			Limit:  &limit,
			Offset: &offset,
		}, "items(track(id,name,artists)),total")

		if err != nil {
			log.Fatal("Failed to get playlist tracks:", err)
		}

		for _, track := range tracksPage.Tracks {
			if track.Track.ID != "" {
				savedSongs[string(track.Track.ID)] = fmt.Sprintf("%s by %s", track.Track.Name, getArtistsNames(track.Track.Artists))
			}
		}

		offset += len(tracksPage.Tracks)

		// Break the loop if we've processed all tracks
		if offset >= tracksPage.Total {
			break
		}
	}

	name := p.Name
	if len(p.Name) > 32 {
		name = p.Name[0:32]
	}

	savedSongsFilename := path.Join(playlistDirectory, fmt.Sprintf("%s-%s-%s.json", savedSongsBaseFilename, name, p.ID))
	if fileExists(savedSongsFilename) {
		previouslySavedSongs, err := readJsonFromFile(savedSongsFilename)
		if err != nil {
			log.Fatalln("Failed to read from file in processPlaylist:", err)
		}

		// note, I'm not using beeep here for missing song notifications, as it would be too much notifying for each playlist
		missingSongs, searchResults := findMissingSongs(previouslySavedSongs, savedSongs)
		if len(missingSongs) > 0 {
			fmt.Println("The following songs are missing from this playlist:")
			for id, song := range missingSongs {
				fmt.Printf("\t- ID: %s, Song: %s\n", id, song)
			}

			if len(searchResults) > 0 {
				fmt.Println("\nSearch results for missing songs:")
				for id, results := range searchResults {
					fmt.Printf("For missing song: %s\n", missingSongs[id])
					for trackID, trackInfo := range results {
						matchIndicator := ""
						if trackID == id {
							matchIndicator = " <--- match"
						}
						fmt.Printf("\t- ID: %s, Song: %s%s\n", trackID, trackInfo, matchIndicator)
					}
				}
			}
		} else {
			fmt.Println("No songs are missing from your", p.Name, "playlist - yay")
		}
	}

	writeJsonToFile(savedSongsFilename, savedSongs)

	fmt.Println()
}

// Function to get the songs of a playlist
// Note, this is not Liked Songs, like the Spotify client shows, its actually just a playlist named Liked Songs
// you need to use SavedTracks
// this has an automatic limit of 100 tracks due to GetPlaylist function, need to use GetPlaylistTracksOpt to get more via pagination
// func getPlaylistSongs(client spotify.Client, playlistId spotify.ID) (*spotify.FullPlaylist, error) {
// 	playlist, err := client.GetPlaylist(playlistId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return playlist, nil
// }

func getSavedTracks(client spotify.Client) ([]spotify.SavedTrack, error) {
	var savedTracks []spotify.SavedTrack

	opt := spotify.Options{Limit: new(int), Offset: new(int)}

	for {
		*opt.Limit = 50
		*opt.Offset = len(savedTracks)

		page, err := client.CurrentUsersTracksOpt(&opt)
		if err != nil {
			return nil, err
		}

		savedTracks = append(savedTracks, page.Tracks...)

		fmt.Printf("\r%d/%d", len(savedTracks), page.Total)
		os.Stdout.Sync() // animate the progress

		if page.Total <= len(savedTracks) {
			break
		}
	}

	fmt.Println() // Move to the next line after the loop

	return savedTracks, nil
}

// Function to concatenate artist names
func getArtistsNames(artists []spotify.SimpleArtist) string {
	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

// Function to write data to a JSON file
func writeJsonToFile(filename string, data interface{}) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Fatal(err)
	}
}

// Read a JSON file and return a map[string]string
func readJsonFromFile(filename string) (map[string]string, error) {
	var result map[string]string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Compares the maps and finds the missing songs based on the song's track ID
// Also performs a search on Spotify for missing songs
func findMissingSongs(s1 map[string]string, s2 map[string]string) (map[string]string, map[string]map[string]string) {
	missingSongs := make(map[string]string)             // Create a map to store the missing songs
	searchResults := make(map[string]map[string]string) // Create a map to store search results

	// Iterate over each song in the first map
	for id, song := range s1 {
		// If the song is not in the second map, add it to the missing songs
		if _, exists := s2[id]; !exists {
			missingSongs[id] = song

			// Perform a search on Spotify using the entire song string
			result, err := client.Search(song, spotify.SearchTypeTrack)
			if err == nil && len(result.Tracks.Tracks) > 0 {
				searchResults[id] = make(map[string]string)
				for _, track := range result.Tracks.Tracks {
					searchResults[id][string(track.ID)] = fmt.Sprintf("%s by %s", track.Name, getArtistsNames(track.Artists))
				}
			}
		}
	}

	return missingSongs, searchResults
}

// Function to get all playlists
func getAllPlaylists(client spotify.Client) ([]spotify.SimplePlaylist, error) {
	var playlists []spotify.SimplePlaylist

	var limit = 50 // Adjust the limit as needed
	opt := &spotify.Options{Limit: &limit}
	offset := 0

	for {
		user, err := client.CurrentUser()
		if err != nil {
			return nil, err
		}

		opt.Offset = &offset
		userPlaylists, err := client.GetPlaylistsForUserOpt(user.ID, opt)
		if err != nil {
			return nil, err
		}

		playlists = append(playlists, userPlaylists.Playlists...)

		if userPlaylists.Next == "" {
			break
		}

		offset += len(userPlaylists.Playlists)
	}

	return playlists, nil
}
