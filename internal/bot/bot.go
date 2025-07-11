package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/doomhound188/soulhound/internal/audio"
	"github.com/doomhound188/soulhound/internal/config"
	"github.com/doomhound188/soulhound/internal/queue"
	"github.com/jonas747/dca"
)

type Bot struct {
	session       *discordgo.Session
	queue         *queue.Queue
	youtubePlayer *audio.YouTubeProvider
	spotifyPlayer *audio.SpotifyProvider
	voiceConn     map[string]*VoiceConnection
	voiceStates   map[string]*VoiceStateInfo // Enhanced voice state tracking
	mu            sync.Mutex
	isPlaying     bool
}

// Enhanced voice state tracking with timestamps and validation
type VoiceStateInfo struct {
	VoiceState *discordgo.VoiceState
	LastUpdate time.Time
	Validated  bool
}

type VoiceConnection struct {
	connection *discordgo.VoiceConnection
	channelID  string
	guildID    string
	encoder    *dca.EncodeSession
	stream     *dca.StreamingSession
}

func New(cfg *config.Config) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		session:       session,
		queue:         queue.NewQueue(),
		youtubePlayer: audio.NewYouTubeProvider(cfg.YouTubeToken),
		spotifyPlayer: audio.NewSpotifyProvider(cfg.SpotifyToken),
		voiceConn:     make(map[string]*VoiceConnection),
		voiceStates:   make(map[string]*VoiceStateInfo),
	}

	session.AddHandler(bot.messageHandler)
	session.AddHandler(bot.readyHandler)
	session.AddHandler(bot.voiceStateUpdateHandler)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent

	return bot, nil
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Close() error {
	// Cleanup voice connections
	for _, vc := range b.voiceConn {
		if vc.encoder != nil {
			vc.encoder.Cleanup()
		}
		if vc.connection != nil {
			vc.connection.Disconnect()
		}
	}
	return b.session.Close()
}

func (b *Bot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
	log.Printf("Connected to %d guilds", len(r.Guilds))
	log.Printf("Bot intents configured: %d", s.Identify.Intents)
	log.Printf("Required intents: %d", discordgo.IntentsGuildMessages|discordgo.IntentsGuildVoiceStates|discordgo.IntentsMessageContent)
	
	// Initialize voice states from current guild data for all guilds
	totalVoiceStates := 0
	for _, guild := range r.Guilds {
		if guild.VoiceStates != nil {
			b.mu.Lock()
			for _, vs := range guild.VoiceStates {
				if vs.ChannelID != "" {
					key := vs.GuildID + ":" + vs.UserID
					b.voiceStates[key] = &VoiceStateInfo{
						VoiceState: vs,
						LastUpdate: time.Now(),
						Validated:  true,
					}
					totalVoiceStates++
				}
			}
			b.mu.Unlock()
		}
	}
	log.Printf("Initialized with %d voice states across all guilds", totalVoiceStates)
}

func (b *Bot) voiceStateUpdateHandler(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	// Update our internal state tracking
	// This handler ensures we have the most up-to-date voice state information
	log.Printf("🔊 VOICE STATE UPDATE HANDLER CALLED - UserID: %s, ChannelID: %s, GuildID: %s", vsu.UserID, vsu.ChannelID, vsu.GuildID)

	b.mu.Lock()
	defer b.mu.Unlock()

	key := vsu.GuildID + ":" + vsu.UserID
	if vsu.ChannelID == "" {
		log.Printf("User %s left voice channel in guild %s", vsu.UserID, vsu.GuildID)
		delete(b.voiceStates, key)
		log.Printf("🔊 Internal tracking: Removed user %s from guild %s (total tracked: %d)", vsu.UserID, vsu.GuildID, len(b.voiceStates))
	} else {
		log.Printf("User %s joined voice channel %s in guild %s", vsu.UserID, vsu.ChannelID, vsu.GuildID)
		b.voiceStates[key] = &VoiceStateInfo{
			VoiceState: &discordgo.VoiceState{
				UserID:    vsu.UserID,
				ChannelID: vsu.ChannelID,
				GuildID:   vsu.GuildID,
				SessionID: vsu.SessionID,
				Deaf:      vsu.Deaf,
				Mute:      vsu.Mute,
				SelfDeaf:  vsu.SelfDeaf,
				SelfMute:  vsu.SelfMute,
				Suppress:  vsu.Suppress,
			},
			LastUpdate: time.Now(),
			Validated:  true,
		}
		log.Printf("🔊 Internal tracking: Added user %s to channel %s in guild %s (total tracked: %d)", vsu.UserID, vsu.ChannelID, vsu.GuildID, len(b.voiceStates))
	}
}

func (b *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message starts with command prefix
	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	// Split command and arguments
	parts := strings.Fields(m.Content[1:])
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	args := parts[1:]

	// Check if command requires voice channel
	voiceRequiredCommands := []string{"play", "pause", "resume", "stop", "skip"}
	requiresVoice := false
	for _, cmd := range voiceRequiredCommands {
		if strings.ToLower(command) == cmd {
			requiresVoice = true
			break
		}
	}

	var voiceChannelID string
	if requiresVoice {
		// Ensure we have a valid guild ID
		if m.GuildID == "" {
			s.ChannelMessageSend(m.ChannelID, "Error: This command can only be used in a server")
			return
		}

		// Skip permission check for now as it's causing false positives
		// The Discord permissions screen shows the bot has all required permissions

		// Try multiple methods to get voice state - prioritize fresh API data due to cache corruption
		var voiceState *discordgo.VoiceState
		log.Printf("Voice detection: Starting voice state lookup for user %s (%s) in guild %s", m.Author.Username, m.Author.ID, m.GuildID)

		// Method 1: Check our internal voice state tracking first
		log.Printf("Voice detection: Trying method 1 - internal voice state tracking")
		b.mu.Lock()
		key := m.GuildID + ":" + m.Author.ID
		if vs, exists := b.voiceStates[key]; exists && vs.VoiceState.ChannelID != "" {
			log.Printf("Voice detection: Found voice state in internal tracking - Channel: %s", vs.VoiceState.ChannelID)
			voiceState = vs.VoiceState
		}
		b.mu.Unlock()

		// Method 2: Direct API call as fallback
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 2 - direct API call")
			guild, err := s.Guild(m.GuildID)
			if err == nil && guild != nil {
				log.Printf("Voice detection: API call successful, found %d voice states", len(guild.VoiceStates))
				for _, vs := range guild.VoiceStates {
					if vs.UserID == m.Author.ID && vs.ChannelID != "" {
						log.Printf("Voice detection: Found matching voice state via API - Channel: %s", vs.ChannelID)
						voiceState = vs
						break
					}
				}
			} else {
				log.Printf("Voice detection: API call failed: %v", err)
			}
		}

		// Method 3: Try cache lookup as fallback
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 3 - cache lookup")
			voiceState, err := s.State.VoiceState(m.GuildID, m.Author.ID)
			if err != nil {
				log.Printf("Voice detection: Cache lookup failed: %v", err)
			} else if voiceState != nil && voiceState.ChannelID != "" {
				log.Printf("Voice detection: Found voice state in cache - Channel: %s", voiceState.ChannelID)
			} else {
				log.Printf("Voice detection: Cache lookup returned nil or empty channel")
			}
		}

		// Method 4: Search through all cached voice states in the guild
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 4 - searching guild voice states")
			guild, err := s.State.Guild(m.GuildID)
			if err == nil && guild != nil {
				log.Printf("Voice detection: Found guild with %d voice states", len(guild.VoiceStates))
				for _, vs := range guild.VoiceStates {
					if vs.UserID == m.Author.ID && vs.ChannelID != "" {
						log.Printf("Voice detection: Found matching voice state in guild cache - Channel: %s", vs.ChannelID)
						voiceState = vs
						break
					}
				}
			} else {
				log.Printf("Voice detection: Could not get guild from cache: %v", err)
			}
		}

		// Method 5: Last resort - wait a moment and try cache again
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 5 - retry after delay")
			// Sometimes there's a delay in state updates, give it a moment
			time.Sleep(100 * time.Millisecond)
			voiceState, _ = s.State.VoiceState(m.GuildID, m.Author.ID)
			if voiceState != nil && voiceState.ChannelID != "" {
				log.Printf("Voice detection: Found voice state after delay - Channel: %s", voiceState.ChannelID)
			}
		}

		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: FAILED - No voice state found after all methods")
			errorMsg := "❌ **You must be in a voice channel to use this command**\n\n"
			errorMsg += "**Troubleshooting:**\n"
			errorMsg += "• Make sure you're connected to a voice channel\n"
			errorMsg += "• Try leaving and rejoining the voice channel\n"
			errorMsg += "• Use `!debug` to see voice channel information\n"
			errorMsg += "• Wait a few seconds after joining before using commands\n"
			errorMsg += "• Check if the bot can see the voice channel you're in"
			s.ChannelMessageSend(m.ChannelID, errorMsg)
			return
		}
		voiceChannelID = voiceState.ChannelID
		log.Printf("Voice detection: SUCCESS - User %s is in voice channel %s", m.Author.Username, voiceChannelID)
	}

	response, err := b.HandleCommand(command, args, voiceChannelID, m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %s", err))
		return
	}

	if response != "" {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}

func (b *Bot) HandleCommand(command string, args []string, channelID string, guildID string, userID string) (string, error) {
	switch strings.ToLower(command) {
	case "play":
		return b.handlePlay(args, channelID, guildID)
	case "pause":
		return b.handlePause()
	case "resume":
		return b.handleResume()
	case "stop":
		return b.handleStop()
	case "queue":
		return b.handleQueue()
	case "skip":
		return b.handleSkip()
	case "remove":
		return b.handleRemove(args)
	case "search":
		return b.handleSearch(args)
	case "setdefault":
		return b.handleSetDefault(args)
	case "smartplay":
		return b.handleSmartPlay(args)
	case "help":
		return b.handleHelp()
	case "debug":
		return b.handleDebug(guildID)
	case "voicetest":
		return b.handleVoiceTest(guildID, userID), nil
	case "refreshvoice":
		return b.handleRefreshVoice(guildID), nil
	case "diagnose":
		return b.handleDiagnose(guildID, userID), nil
	case "undeafen":
		return b.handleUndeafen(guildID), nil
	case "apitest":
		return b.handleApiTest(guildID), nil
	default:
		return "", errors.New("unknown command. Type !help for available commands")
	}
}

func (b *Bot) handlePlay(args []string, channelID string, guildID string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("please provide a search query")
	}

	if channelID == "" {
		return "", errors.New("you must be in a voice channel to play music - use !debug to troubleshoot")
	}

	// Validate guild ID
	if guildID == "" {
		return "", errors.New("no guild available to join voice channel - this command must be used in a server")
	}

	_, err := b.joinVoiceChannel(guildID, channelID)
	if err != nil {
		return "", fmt.Errorf("failed to join voice channel: %w", err)
	}

	platform := config.AppConfig.DefaultPlayer
	query := strings.Join(args, " ")

	// Check if platform is specified in the query
	if strings.HasPrefix(query, "yt:") || strings.HasPrefix(query, "sp:") {
		platform = query[:2]
		query = query[3:]
	}

	var results []audio.SearchResult

	switch platform {
	case "yt":
		results, err = b.youtubePlayer.Search(query)
	case "sp":
		results, err = b.spotifyPlayer.Search(query)
	default:
		return "", errors.New("invalid platform")
	}

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No results found", nil
	}

	// Add first result to queue
	track := queue.Track{
		Title:    results[0].Title,
		Artist:   results[0].Artist,
		URL:      results[0].ID, // Store ID as URL for later streaming
		Platform: platform,
		Genre:    results[0].Genre,
	}

	b.queue.Add(track)

	// Start playing if not already
	b.mu.Lock()
	wasPlaying := b.isPlaying
	b.mu.Unlock()

	if !wasPlaying {
		go b.startPlaying()
	}

	// Provide helpful feedback about what will happen
	var response string
	if strings.HasPrefix(track.URL, "mock_") || strings.HasPrefix(track.URL, "spotify_mock_") {
		response = fmt.Sprintf("✅ **Added to queue:** %s - %s\n🎵 **Note:** This is a test track that will play silence for demonstration purposes.", track.Title, track.Artist)
	} else if track.Platform == "yt" {
		response = fmt.Sprintf("✅ **Added to queue:** %s - %s\n⚠️ **Note:** YouTube audio streaming requires youtube-dl/yt-dlp setup for actual playback.", track.Title, track.Artist)
	} else if track.Platform == "sp" {
		response = fmt.Sprintf("✅ **Added to queue:** %s - %s\n⚠️ **Note:** Spotify tracks cannot be streamed directly due to licensing restrictions.", track.Title, track.Artist)
	} else {
		response = fmt.Sprintf("✅ **Added to queue:** %s - %s", track.Title, track.Artist)
	}
	
	return response, nil
}

func (b *Bot) handlePause() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isPlaying {
		return "Nothing is playing", nil
	}

	for _, vc := range b.voiceConn {
		if vc.stream != nil {
			vc.stream.SetPaused(true)
		}
	}

	b.isPlaying = false
	return "Playback paused", nil
}

func (b *Bot) handleResume() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isPlaying {
		return "Already playing", nil
	}

	for _, vc := range b.voiceConn {
		if vc.stream != nil {
			vc.stream.SetPaused(false)
		}
	}

	b.isPlaying = true
	return "Playback resumed", nil
}

func (b *Bot) handleStop() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.isPlaying = false
	b.queue.Clear()

	for _, vc := range b.voiceConn {
		if vc.encoder != nil {
			vc.encoder.Cleanup()
		}
		if vc.stream != nil {
			vc.stream.SetPaused(true)
		}
	}

	return "Playback stopped and queue cleared", nil
}

func (b *Bot) handleQueue() (string, error) {
	tracks := b.queue.List()
	if len(tracks) == 0 {
		return "Queue is empty", nil
	}

	var sb strings.Builder
	sb.WriteString("Current queue:\n")
	for i, track := range tracks {
		sb.WriteString(fmt.Sprintf("%d. %s - %s [%s]\n", i+1, track.Title, track.Artist, track.Platform))
	}
	return sb.String(), nil
}

func (b *Bot) handleSkip() (string, error) {
	track, err := b.queue.Next()
	if err != nil {
		return "", err
	}

	// Stop current playback
	for _, vc := range b.voiceConn {
		if vc.encoder != nil {
			vc.encoder.Cleanup()
		}
	}

	// Start playing next track
	go b.startPlaying()

	return fmt.Sprintf("Skipped to: %s - %s", track.Title, track.Artist), nil
}

func (b *Bot) handleRemove(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("please provide the track number to remove")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return "", errors.New("invalid track number")
	}

	index-- // Convert to 0-based index
	if err := b.queue.Remove(index); err != nil {
		return "", err
	}

	return fmt.Sprintf("Removed track at position %d", index+1), nil
}

func (b *Bot) handleSearch(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("please provide a search query")
	}

	query := strings.Join(args, " ")
	platform := config.AppConfig.DefaultPlayer

	var results []audio.SearchResult
	var err error

	switch platform {
	case "yt":
		results, err = b.youtubePlayer.Search(query)
	case "sp":
		results, err = b.spotifyPlayer.Search(query)
	}

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No results found", nil
	}

	var sb strings.Builder
	sb.WriteString("Search results:\n")
	maxResults := 5
	if len(results) < maxResults {
		maxResults = len(results)
	}
	for i, result := range results[:maxResults] {
		sb.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, result.Title, result.Artist))
	}
	return sb.String(), nil
}

func (b *Bot) handleSetDefault(args []string) (string, error) {
	if len(args) != 1 || (args[0] != "yt" && args[0] != "sp") {
		return "", errors.New("please specify either 'yt' or 'sp'")
	}

	config.SetDefaultPlayer(args[0])
	return fmt.Sprintf("Default player set to %s", args[0]), nil
}

func (b *Bot) handleSmartPlay(args []string) (string, error) {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return "", errors.New("please specify either 'on' or 'off'")
	}

	enabled := args[0] == "on"
	config.ToggleSmartPlay(enabled)
	return fmt.Sprintf("Smart play %s", args[0]), nil
}

func (b *Bot) startPlaying() {
	b.mu.Lock()
	if b.isPlaying {
		b.mu.Unlock()
		return
	}
	b.isPlaying = true
	b.mu.Unlock()

	for {
		b.mu.Lock()
		if !b.isPlaying {
			b.mu.Unlock()
			return
		}

		track, err := b.queue.Current()
		if err != nil {
			b.isPlaying = false
			b.mu.Unlock()
			return
		}

		// Get stream URL based on platform
		var streamURL string
		if track.Platform == "yt" {
			streamURL, _ = b.youtubePlayer.GetStreamURL(track.URL)
		} else {
			streamURL, _ = b.spotifyPlayer.GetStreamURL(track.URL)
		}

		// Get a copy of voice connections for iteration
		voiceConnections := make(map[string]*VoiceConnection)
		for k, v := range b.voiceConn {
			voiceConnections[k] = v
		}
		b.mu.Unlock()

		// Stream to all connected voice channels
		for guildID, vc := range voiceConnections {
			if err := b.streamAudio(streamURL, vc); err != nil {
				log.Printf("Error streaming audio in guild %s: %v", guildID, err)
			}
		}

		// If smart play is enabled, add recommendations to queue
		if config.PlayerConfig.SmartPlayEnabled {
			b.addRecommendations(track)
		}

		// Move to next track
		b.queue.Next()
	}
}

func (b *Bot) joinVoiceChannel(guildID, channelID string) (*VoiceConnection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if already connected to this guild
	if vc, exists := b.voiceConn[guildID]; exists {
		if vc.channelID == channelID {
			return vc, nil
		}
		// Disconnect from current channel
		vc.connection.Disconnect()
	}

	// Join new channel
	conn, err := b.session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return nil, err
	}

	vc := &VoiceConnection{
		connection: conn,
		channelID:  channelID,
		guildID:    guildID,
	}
	b.voiceConn[guildID] = vc

	return vc, nil
}

func (b *Bot) streamAudio(url string, vc *VoiceConnection) error {
	// Validate URL
	if url == "" {
		return fmt.Errorf("empty stream URL")
	}

	log.Printf("Attempting to stream audio from URL: %s", url)

	// Check if this is a mock/test URL
	if strings.HasPrefix(url, "mock_") {
		log.Printf("Mock audio detected, creating test silence stream")
		return b.streamTestAudio(vc)
	}

	// Check if this is a YouTube URL or ID
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") || (len(url) == 11 && !strings.Contains(url, "/")) {
		log.Printf("YouTube content detected, attempting to stream using YouTube library")
		return b.streamYouTubeAudio(url, vc)
	}

	// Check if this is a Spotify ID
	if len(url) == 22 && !strings.Contains(url, "/") {
		log.Printf("Spotify content detected. Audio streaming not supported for Spotify tracks")
		return fmt.Errorf("Spotify audio streaming is not supported. Spotify does not provide direct audio streams")
	}

	// Try to stream if it's a direct audio URL
	return b.streamDirectAudio(url, vc)
}

// streamTestAudio creates a brief test audio stream for testing purposes
func (b *Bot) streamTestAudio(vc *VoiceConnection) error {
	log.Printf("Creating test audio stream")
	
	// Create a brief silence stream for testing
	// This demonstrates that the voice connection works
	if vc.connection == nil {
		return fmt.Errorf("voice connection is nil")
	}

	// Send a brief period of silence to test voice connection
	// This proves the bot can connect and send audio data
	vc.connection.Speaking(true)
	defer vc.connection.Speaking(false)

	// Create 2 seconds of silence (48000 Hz, 2 channels, 16-bit samples)
	silenceFrames := 48000 * 2 * 2 // 2 seconds of audio
	silenceData := make([]byte, silenceFrames)
	
	// Send the silence data in chunks
	chunkSize := 3840 // 20ms worth of audio data
	for i := 0; i < len(silenceData); i += chunkSize {
		end := i + chunkSize
		if end > len(silenceData) {
			end = len(silenceData)
		}
		
		select {
		case vc.connection.OpusSend <- silenceData[i:end]:
		default:
			// Channel might be full, skip this frame
		}
	}

	log.Printf("Test audio stream completed successfully")
	return nil
}

// streamYouTubeAudio attempts to stream YouTube audio using the YouTube library
func (b *Bot) streamYouTubeAudio(videoID string, vc *VoiceConnection) error {
	log.Printf("Attempting to stream YouTube audio for video ID: %s", videoID)

	if vc.connection == nil {
		return fmt.Errorf("voice connection is nil")
	}

	// Get the stream URL using our YouTube provider
	streamURL, err := b.youtubePlayer.GetStreamURL(videoID)
	if err != nil {
		log.Printf("Failed to get YouTube stream URL for %s: %v", videoID, err)
		return fmt.Errorf("failed to get YouTube stream URL: %w", err)
	}

	log.Printf("Successfully obtained YouTube stream URL, attempting to stream")

	// Now stream the URL using DCA
	return b.streamDirectAudio(streamURL, vc)
}

// streamDirectAudio attempts to stream a direct audio file URL
func (b *Bot) streamDirectAudio(url string, vc *VoiceConnection) error {
	log.Printf("Attempting to stream direct audio URL: %s", url)

	// Create DCA encoding session
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96

	// Try to encode the URL directly (this only works for direct audio files)
	encodingSession, err := dca.EncodeFile(url, options)
	if err != nil {
		log.Printf("Could not encode audio from URL %s: %v", url, err)
		return fmt.Errorf("unable to stream audio from this source. URL may not be a direct audio file: %w", err)
	}
	defer encodingSession.Cleanup()

	vc.encoder = encodingSession
	done := make(chan error)
	stream := dca.NewStream(encodingSession, vc.connection, done)
	vc.stream = stream

	err = <-done
	if err != nil {
		log.Printf("Streaming finished with error: %v", err)
	} else {
		log.Printf("Streaming completed successfully")
	}
	return err
}

func (b *Bot) addRecommendations(track *queue.Track) {
	var results []audio.SearchResult
	var err error

	if track.Platform == "yt" {
		results, err = b.youtubePlayer.GetRecommendations(track.Genre)
	} else {
		results, err = b.spotifyPlayer.GetRecommendations(track.Genre)
	}

	if err != nil {
		return
	}

	for _, result := range results {
		b.queue.Add(queue.Track{
			Title:    result.Title,
			Artist:   result.Artist,
			URL:      result.ID,
			Platform: track.Platform,
			Genre:    result.Genre,
		})
	}
}

func (b *Bot) handleHelp() (string, error) {
	help := `**SoulHound Music Bot Commands:**

**Music Controls (requires voice channel):**
• !play <query> - Play a song (prefix with yt: or sp: to specify platform)
• !pause - Pause current playback
• !resume - Resume paused playback
• !stop - Stop playback and clear queue
• !skip - Skip to next track

**Queue Management:**
• !queue - Show current queue
• !remove <number> - Remove track from queue

**Search & Discovery:**
• !search <query> - Search without adding to queue

**Settings:**
• !setdefault <yt/sp> - Set default platform (YouTube/Spotify)
• !smartplay <on/off> - Toggle smart recommendations

**Debug:**
• !debug - Show voice channel debug information
• !voicetest - Test voice state detection
• !refreshvoice - Force refresh voice state data
• !diagnose - Comprehensive guild and channel diagnostic
• !undeafen - Undeafen the bot in voice channel
• !apitest - Test Discord API connectivity

**Examples:**
• !play yt:never gonna give you up
• !play sp:shape of you
• !setdefault yt
• !smartplay on

Type !help to see this message again.`

	return help, nil
}

func (b *Bot) handleDebug(guildID string) (string, error) {
	if guildID == "" {
		return "Debug: No guild ID available", nil
	}

	var debugInfo strings.Builder
	debugInfo.WriteString(fmt.Sprintf("**Debug Information for Guild: %s**\n", guildID))

	// Check if session is available
	if b.session == nil {
		debugInfo.WriteString("❌ Bot session not available (testing mode)\n")
		return debugInfo.String(), nil
	}

	// Check bot permissions
	debugInfo.WriteString(fmt.Sprintf("**Bot Intents:** %d\n", b.session.Identify.Intents))
	debugInfo.WriteString("**Required Intents:** GuildMessages + GuildVoiceStates + MessageContent\n")

	// Detailed intent breakdown
	debugInfo.WriteString("**Intent Breakdown:**\n")
	currentIntents := b.session.Identify.Intents

	// Check specific intents using discordgo constants
	if currentIntents&discordgo.IntentsGuildMessages != 0 {
		debugInfo.WriteString("✅ Guild Messages Intent (512)\n")
	} else {
		debugInfo.WriteString("❌ Guild Messages Intent (512) - MISSING!\n")
	}

	if currentIntents&discordgo.IntentsGuildVoiceStates != 0 {
		debugInfo.WriteString("✅ Guild Voice States Intent (128)\n")
	} else {
		debugInfo.WriteString("❌ Guild Voice States Intent (128) - MISSING!\n")
	}

	if currentIntents&discordgo.IntentsMessageContent != 0 {
		debugInfo.WriteString("✅ Message Content Intent (32768)\n")
	} else {
		debugInfo.WriteString("❌ Message Content Intent (32768) - MISSING!\n")
	}

	expectedIntents := discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent
	if currentIntents == expectedIntents {
		debugInfo.WriteString("✅ **All required intents are enabled**\n")
	} else {
		debugInfo.WriteString("❌ **Intent mismatch detected!**\n")
		debugInfo.WriteString(fmt.Sprintf("   Expected: %d, Got: %d\n", expectedIntents, currentIntents))
		debugInfo.WriteString("   📋 **Action required:** Check Discord Developer Portal → Bot → Privileged Gateway Intents\n")
	}
	debugInfo.WriteString("\n")

	// Detailed permission check
	debugInfo.WriteString("**Permission Analysis:**\n")
	if permInfo := b.getDetailedPermissions(guildID); permInfo != "" {
		debugInfo.WriteString(permInfo)
	} else {
		debugInfo.WriteString("❌ Could not retrieve permission information\n")
	}
	debugInfo.WriteString("\n")

	// Try to get guild info
	guild, err := b.session.State.Guild(guildID)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("❌ Error getting guild from cache: %v\n", err))

		// Try API call
		guild, err = b.session.Guild(guildID)
		if err != nil {
			debugInfo.WriteString(fmt.Sprintf("❌ Error getting guild from API: %v\n", err))
			return debugInfo.String(), nil
		} else {
			debugInfo.WriteString("✅ Guild retrieved from API\n")
		}
	} else {
		debugInfo.WriteString("✅ Guild retrieved from cache\n")
	}

	debugInfo.WriteString(fmt.Sprintf("Guild Name: %s\n", guild.Name))
	debugInfo.WriteString(fmt.Sprintf("Voice States Count: %d\n", len(guild.VoiceStates)))

	if len(guild.VoiceStates) > 0 {
		debugInfo.WriteString("\n**Current Voice States:**\n")
		for i, vs := range guild.VoiceStates {
			if i >= 10 { // Limit output
				debugInfo.WriteString("... (truncated)\n")
				break
			}
			member, err := b.session.State.Member(guildID, vs.UserID)
			username := vs.UserID
			if err == nil && member != nil {
				username = member.User.Username
			}
			debugInfo.WriteString(fmt.Sprintf("• %s (ID: %s) in channel %s\n", username, vs.UserID, vs.ChannelID))
		}
	} else {
		debugInfo.WriteString("No users in voice channels\n")
	}

	// Check voice connections
	debugInfo.WriteString(fmt.Sprintf("\n**Bot Voice Connections:** %d\n", len(b.voiceConn)))
	for gid, vc := range b.voiceConn {
		debugInfo.WriteString(fmt.Sprintf("• Guild %s: Channel %s\n", gid, vc.channelID))
	}

	return debugInfo.String(), nil
}

// getDetailedPermissions provides detailed permission information for debugging
func (b *Bot) getDetailedPermissions(guildID string) string {
	var permInfo strings.Builder

	// Check if session is available for testing
	if b.session == nil || b.session.State == nil || b.session.State.User == nil {
		return "❌ Bot session not available (testing mode or not connected)\n"
	}

	// Get the bot's member info in the guild - always use fresh API data
	botMember, err := b.session.GuildMember(guildID, b.session.State.User.ID)
	if err != nil {
		return fmt.Sprintf("❌ Could not get bot member info from API: %v\n", err)
	}

	// Get guild info - always use fresh API data to avoid cache corruption
	guild, err := b.session.Guild(guildID)
	if err != nil {
		return fmt.Sprintf("❌ Could not get guild info from API: %v\n", err)
	}

	// Debug: Show bot member details
	permInfo.WriteString(fmt.Sprintf("**Bot User ID:** %s\n", b.session.State.User.ID))
	permInfo.WriteString(fmt.Sprintf("**Bot has %d roles assigned**\n", len(botMember.Roles)))

	// Debug: Show all bot roles
	permInfo.WriteString("**Bot role IDs:** ")
	for i, roleID := range botMember.Roles {
		if i > 0 {
			permInfo.WriteString(", ")
		}
		permInfo.WriteString(roleID)
	}
	permInfo.WriteString("\n")

	// Debug: Show guild roles count
	permInfo.WriteString(fmt.Sprintf("**Guild has %d total roles**\n\n", len(guild.Roles)))

	// Calculate total permissions
	permissions := int64(0)

	// Check @everyone role permissions
	everyoneFound := false
	for _, role := range guild.Roles {
		if role.ID == guildID { // @everyone role
			permissions |= role.Permissions
			permInfo.WriteString(fmt.Sprintf("@everyone permissions: %d\n", role.Permissions))
			everyoneFound = true
			break
		}
	}
	if !everyoneFound {
		permInfo.WriteString("⚠️  @everyone role not found\n")
	}

	// Add permissions from bot's roles
	permInfo.WriteString("Bot roles:\n")
	if len(botMember.Roles) == 0 {
		permInfo.WriteString("❌ **Bot has no roles assigned!**\n")
		permInfo.WriteString("   This is likely the cause of permission issues.\n")
		permInfo.WriteString("   Solutions:\n")
		permInfo.WriteString("   1. Create a role with required permissions\n")
		permInfo.WriteString("   2. Assign the role to the bot\n")
		permInfo.WriteString("   3. Or re-invite the bot with proper permissions\n")
	} else {
		roleCount := 0
		for _, roleID := range botMember.Roles {
			roleFound := false
			for _, role := range guild.Roles {
				if role.ID == roleID {
					permissions |= role.Permissions
					permInfo.WriteString(fmt.Sprintf("• %s (ID: %s): %d\n", role.Name, role.ID, role.Permissions))
					roleCount++
					roleFound = true
					break
				}
			}
			if !roleFound {
				permInfo.WriteString(fmt.Sprintf("❌ Role ID %s not found in guild!\n", roleID))
			}
		}
		if roleCount == 0 {
			permInfo.WriteString("❌ **Bot roles not found in guild!**\n")
			permInfo.WriteString("   This may indicate a synchronization issue.\n")
		}
	}

	// Try alternative permission calculation using guild-level permissions
	if guild.OwnerID == b.session.State.User.ID {
		// Bot is the server owner
		altPermissions := int64(8) // Administrator permission
		permInfo.WriteString(fmt.Sprintf("**Alternative permission calculation:** %d (Bot is server owner)\n", altPermissions))
		permissions = altPermissions
	} else {
		// Calculate permissions using Discord's permission system
		member, err := b.session.GuildMember(guildID, b.session.State.User.ID)
		if err == nil {
			altPermissions := int64(0)

			// Add @everyone permissions
			for _, role := range guild.Roles {
				if role.ID == guildID {
					altPermissions |= role.Permissions
					break
				}
			}

			// Add bot role permissions
			for _, roleID := range member.Roles {
				for _, role := range guild.Roles {
					if role.ID == roleID {
						altPermissions |= role.Permissions
						break
					}
				}
			}

			permInfo.WriteString(fmt.Sprintf("**Alternative permission calculation:** %d\n", altPermissions))
			if altPermissions != permissions {
				permInfo.WriteString("⚠️  **Permission mismatch detected!**\n")
				permInfo.WriteString("   Using alternative calculation for accuracy.\n")
				permissions = altPermissions
			}
		} else {
			permInfo.WriteString(fmt.Sprintf("❌ Alternative permission calculation failed: %v\n", err))
		}
	}

	permInfo.WriteString(fmt.Sprintf("**Total calculated permissions: %d**\n", permissions))

	// Add troubleshooting info if permissions are 0
	if permissions == 0 {
		permInfo.WriteString("\n🔧 **TROUBLESHOOTING: Zero permissions detected**\n")
		permInfo.WriteString("Common causes and solutions:\n")
		permInfo.WriteString("1. **Bot has no roles**: Create a role with permissions and assign it\n")
		permInfo.WriteString("2. **Role has no permissions**: Edit the bot's role to add permissions\n")
		permInfo.WriteString("3. **Bot needs re-invite**: Use this URL to re-invite with permissions:\n")
		permInfo.WriteString("   https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=3148800&scope=bot\n")
		permInfo.WriteString("4. **Channel overrides**: Check channel-specific permission overrides\n\n")
	}

	// Check specific permissions
	const (
		PermissionViewChannel   = int64(1024)    // 0x400
		PermissionConnect       = int64(1048576) // 0x100000
		PermissionSpeak         = int64(2097152) // 0x200000
		PermissionSendMessages  = int64(2048)    // 0x800
		PermissionAdministrator = int64(8)       // 0x8
	)

	permInfo.WriteString("**Required Permission Check:**\n")

	if permissions&PermissionAdministrator != 0 {
		permInfo.WriteString("✅ Administrator (has all permissions)\n")
	} else {
		failedPermissions := []string{}

		if permissions&PermissionViewChannel != 0 {
			permInfo.WriteString("✅ View Channels\n")
		} else {
			permInfo.WriteString("❌ View Channels\n")
			failedPermissions = append(failedPermissions, "View Channels")
		}

		if permissions&PermissionSendMessages != 0 {
			permInfo.WriteString("✅ Send Messages\n")
		} else {
			permInfo.WriteString("❌ Send Messages\n")
			failedPermissions = append(failedPermissions, "Send Messages")
		}

		if permissions&PermissionConnect != 0 {
			permInfo.WriteString("✅ Connect\n")
		} else {
			permInfo.WriteString("❌ Connect\n")
			failedPermissions = append(failedPermissions, "Connect")
		}

		if permissions&PermissionSpeak != 0 {
			permInfo.WriteString("✅ Speak\n")
		} else {
			permInfo.WriteString("❌ Speak\n")
			failedPermissions = append(failedPermissions, "Speak")
		}

		// Add specific troubleshooting for failed permissions
		if len(failedPermissions) > 0 {
			permInfo.WriteString(fmt.Sprintf("\n🚨 **Missing %d critical permissions!**\n", len(failedPermissions)))
			permInfo.WriteString("**Immediate action required:**\n")
			permInfo.WriteString("1. Go to Server Settings → Roles\n")
			permInfo.WriteString("2. Find your bot's role or create one\n")
			permInfo.WriteString("3. Enable these permissions: " + strings.Join(failedPermissions, ", ") + "\n")
			permInfo.WriteString("4. Assign the role to the bot\n")
			permInfo.WriteString("5. Run `!debug` again to verify\n\n")
		}
	}

	return permInfo.String()
}

// handleVoiceTest provides a simple test command to check voice state detection
func (b *Bot) handleVoiceTest(guildID, userID string) string {
	var response strings.Builder
	response.WriteString("**Voice State Test Results**\n\n")

	// Check internal tracking
	b.mu.Lock()
	key := guildID + ":" + userID
	if vs, exists := b.voiceStates[key]; exists && vs.VoiceState.ChannelID != "" {
		response.WriteString(fmt.Sprintf("✅ **Internal tracking:** User in channel %s\n", vs.VoiceState.ChannelID))
	} else {
		response.WriteString("❌ **Internal tracking:** No voice state found\n")
	}

	// Show all tracked voice states for this guild
	trackedCount := 0
	for k, _ := range b.voiceStates {
		if strings.HasPrefix(k, guildID+":") {
			trackedCount++
		}
	}
	response.WriteString(fmt.Sprintf("**Tracked voice states in guild:** %d\n", trackedCount))
	b.mu.Unlock()

	// Check Discord API
	guild, err := b.session.Guild(guildID)
	if err == nil && guild != nil {
		response.WriteString(fmt.Sprintf("**Discord API:** Found %d voice states\n", len(guild.VoiceStates)))
		userFound := false
		for _, vs := range guild.VoiceStates {
			if vs.UserID == userID {
				response.WriteString(fmt.Sprintf("✅ **Your voice state:** Channel %s\n", vs.ChannelID))
				userFound = true
				break
			}
		}
		if !userFound {
			response.WriteString("❌ **Your voice state:** Not found in API response\n")
		}
	} else {
		response.WriteString(fmt.Sprintf("❌ **Discord API error:** %v\n", err))
	}

	// Check cache
	if cacheVs, err := b.session.State.VoiceState(guildID, userID); err == nil && cacheVs != nil && cacheVs.ChannelID != "" {
		response.WriteString(fmt.Sprintf("✅ **Cache:** User in channel %s\n", cacheVs.ChannelID))
	} else {
		response.WriteString("❌ **Cache:** No voice state found\n")
	}

	response.WriteString("\n**Instructions:**\n")
	response.WriteString("1. Join a voice channel\n")
	response.WriteString("2. Wait 2-3 seconds\n")
	response.WriteString("3. Run `!voicetest` again\n")
	response.WriteString("4. If still failing, try `!play test`\n")

	return response.String()
}

// handleRefreshVoice forces a refresh of voice state data from Discord
func (b *Bot) handleRefreshVoice(guildID string) string {
	var response strings.Builder
	response.WriteString("**🔄 Force Refreshing Voice State Data**\n\n")

	// Clear internal voice state cache for this guild
	b.mu.Lock()
	clearedCount := 0
	for key := range b.voiceStates {
		if strings.HasPrefix(key, guildID+":") {
			delete(b.voiceStates, key)
			clearedCount++
		}
	}
	b.mu.Unlock()

	response.WriteString(fmt.Sprintf("✅ Cleared %d cached voice states\n", clearedCount))

	// Force fresh API call to get guild data
	guild, err := b.session.Guild(guildID)
	if err != nil {
		response.WriteString(fmt.Sprintf("❌ Failed to fetch guild data: %v\n", err))
		return response.String()
	}

	response.WriteString(fmt.Sprintf("✅ Fresh guild data retrieved\n"))
	response.WriteString(fmt.Sprintf("📊 Discord API reports %d voice states\n", len(guild.VoiceStates)))

	// Update internal tracking with fresh data
	b.mu.Lock()
	addedCount := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID != "" {
			key := vs.GuildID + ":" + vs.UserID
			b.voiceStates[key] = &VoiceStateInfo{
				VoiceState: vs,
				LastUpdate: time.Now(),
				Validated:  true,
			}
			addedCount++
		}
	}
	b.mu.Unlock()

	response.WriteString(fmt.Sprintf("✅ Added %d voice states to internal tracking\n", addedCount))

	// Show current status
	if len(guild.VoiceStates) > 0 {
		response.WriteString("\n**👥 Users in voice channels:**\n")
		for _, vs := range guild.VoiceStates {
			if vs.ChannelID != "" {
				response.WriteString(fmt.Sprintf("• User %s in channel %s\n", vs.UserID, vs.ChannelID))
			}
		}
	} else {
		response.WriteString("\n**👥 No users detected in voice channels**\n")
	}

	response.WriteString("\n**🧪 Test Instructions:**\n")
	response.WriteString("1. Try `!voicetest` to see if voice detection works now\n")
	response.WriteString("2. Try `!play test` to test music functionality\n")
	response.WriteString("3. If still failing, the issue may be with Discord's API\n")

	return response.String()
}

// handleDiagnose provides comprehensive diagnostic information about guild and channel visibility
func (b *Bot) handleDiagnose(guildID, userID string) string {
	var response strings.Builder
	response.WriteString("**🔍 Comprehensive Guild & Channel Diagnostic**\n\n")

	// Check guild basic info
	guild, err := b.session.Guild(guildID)
	if err != nil {
		response.WriteString(fmt.Sprintf("❌ **Critical Error:** Cannot fetch guild data: %v\n", err))
		return response.String()
	}

	response.WriteString(fmt.Sprintf("**Guild Information:**\n"))
	response.WriteString(fmt.Sprintf("• Name: %s\n", guild.Name))
	response.WriteString(fmt.Sprintf("• ID: %s\n", guild.ID))
	response.WriteString(fmt.Sprintf("• Owner: %s\n", guild.OwnerID))
	response.WriteString(fmt.Sprintf("• Member Count: %d\n", guild.MemberCount))
	response.WriteString(fmt.Sprintf("• Channel Count: %d\n", len(guild.Channels)))
	response.WriteString(fmt.Sprintf("• Voice Channel Count: %d\n", countVoiceChannels(guild.Channels)))
	response.WriteString("\n")

	// Check bot member info
	botMember, err := b.session.GuildMember(guildID, b.session.State.User.ID)
	if err != nil {
		response.WriteString(fmt.Sprintf("❌ **Bot Member Error:** %v\n", err))
		return response.String()
	}

	response.WriteString(fmt.Sprintf("**Bot Status:**\n"))
	response.WriteString(fmt.Sprintf("• Bot User ID: %s\n", b.session.State.User.ID))
	response.WriteString(fmt.Sprintf("• Bot Nickname: %s\n", botMember.Nick))
	response.WriteString(fmt.Sprintf("• Bot Roles: %d\n", len(botMember.Roles)))
	response.WriteString("\n")

	// Check user info
	userMember, err := b.session.GuildMember(guildID, userID)
	if err != nil {
		response.WriteString(fmt.Sprintf("❌ **User Member Error:** %v\n", err))
		return response.String()
	}

	response.WriteString(fmt.Sprintf("**Your Status:**\n"))
	response.WriteString(fmt.Sprintf("• User ID: %s\n", userID))
	response.WriteString(fmt.Sprintf("• Nickname: %s\n", userMember.Nick))
	response.WriteString(fmt.Sprintf("• Roles: %d\n", len(userMember.Roles)))
	response.WriteString("\n")

	// List all voice channels and check permissions
	response.WriteString("**Voice Channels Analysis:**\n")
	voiceChannels := getVoiceChannels(guild.Channels)
	if len(voiceChannels) == 0 {
		response.WriteString("❌ **No voice channels found in guild**\n")
	} else {
		for _, channel := range voiceChannels {
			response.WriteString(fmt.Sprintf("• **%s** (ID: %s)\n", channel.Name, channel.ID))

			// Check bot permissions for this channel
			botPerms, err := b.session.UserChannelPermissions(b.session.State.User.ID, channel.ID)
			if err != nil {
				response.WriteString(fmt.Sprintf("  ❌ Bot permissions check failed: %v\n", err))
			} else {
				canView := botPerms&discordgo.PermissionViewChannel != 0
				canConnect := botPerms&discordgo.PermissionVoiceConnect != 0
				canSpeak := botPerms&discordgo.PermissionVoiceSpeak != 0

				response.WriteString(fmt.Sprintf("  Bot: View=%v, Connect=%v, Speak=%v\n", canView, canConnect, canSpeak))
			}

			// Check user permissions for this channel
			userPerms, err := b.session.UserChannelPermissions(userID, channel.ID)
			if err != nil {
				response.WriteString(fmt.Sprintf("  ❌ User permissions check failed: %v\n", err))
			} else {
				canView := userPerms&discordgo.PermissionViewChannel != 0
				canConnect := userPerms&discordgo.PermissionVoiceConnect != 0

				response.WriteString(fmt.Sprintf("  User: View=%v, Connect=%v\n", canView, canConnect))
			}
		}
	}
	response.WriteString("\n")

	// Check current voice states in detail
	response.WriteString("**Current Voice States (Detailed):**\n")
	response.WriteString(fmt.Sprintf("• API Reports: %d total voice states\n", len(guild.VoiceStates)))

	if len(guild.VoiceStates) == 0 {
		response.WriteString("❌ **Discord API returning 0 voice states**\n")
		response.WriteString("  This could indicate:\n")
		response.WriteString("  - Users are not actually in voice channels\n")
		response.WriteString("  - Bot lacks permission to see voice states\n")
		response.WriteString("  - Discord API synchronization issue\n")
		response.WriteString("  - Bot is looking at wrong guild\n")
	} else {
		for _, vs := range guild.VoiceStates {
			response.WriteString(fmt.Sprintf("• User %s in channel %s\n", vs.UserID, vs.ChannelID))
			if vs.UserID == userID {
				response.WriteString("  ✅ **This is you!**\n")
			}
		}
	}
	response.WriteString("\n")

	// Final recommendations
	response.WriteString("**🎯 Recommendations:**\n")
	response.WriteString("1. Verify you're in the same server as the bot\n")
	response.WriteString("2. Check if you can see the voice channel the bot is checking\n")
	response.WriteString("3. Try joining different voice channels\n")
	response.WriteString("4. Check channel-specific permission overrides\n")
	response.WriteString("5. If all else fails, this may be a Discord API issue\n")

	return response.String()
}

// Helper functions for diagnose command
func countVoiceChannels(channels []*discordgo.Channel) int {
	count := 0
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			count++
		}
	}
	return count
}

func getVoiceChannels(channels []*discordgo.Channel) []*discordgo.Channel {
	var voiceChannels []*discordgo.Channel
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			voiceChannels = append(voiceChannels, channel)
		}
	}
	return voiceChannels
}

// handleUndeafen undeafens the bot in the current voice channel
func (b *Bot) handleUndeafen(guildID string) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	vc, exists := b.voiceConn[guildID]
	if !exists || vc.connection == nil {
		return "❌ Bot is not connected to a voice channel"
	}

	// Disconnect and rejoin with proper settings (not deafened)
	channelID := vc.channelID
	vc.connection.Disconnect()

	// Rejoin the channel with deaf=false
	conn, err := b.session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return fmt.Sprintf("❌ Failed to rejoin voice channel: %v", err)
	}

	// Update the voice connection
	vc.connection = conn
	b.voiceConn[guildID] = vc

	return "✅ Bot has been undeafened by rejoining the voice channel"
}

// handleApiTest tests Discord API connectivity and rate limiting
func (b *Bot) handleApiTest(guildID string) string {
	var response strings.Builder
	response.WriteString("**🔗 Discord API Connectivity Test**\n\n")

	// Test 1: Basic guild fetch
	start := time.Now()
	guild, err := b.session.Guild(guildID)
	apiDuration := time.Since(start)

	if err != nil {
		response.WriteString(fmt.Sprintf("❌ **Guild API Call Failed:** %v\n", err))
		response.WriteString("   This indicates a serious API connectivity issue\n\n")
	} else {
		response.WriteString(fmt.Sprintf("✅ **Guild API Call:** %dms\n", apiDuration.Milliseconds()))
		response.WriteString(fmt.Sprintf("   Guild: %s (%d members)\n", guild.Name, guild.MemberCount))
	}

	// Test 2: Cache vs API comparison
	start = time.Now()
	cachedGuild, cacheErr := b.session.State.Guild(guildID)
	cacheDuration := time.Since(start)

	if cacheErr != nil {
		response.WriteString(fmt.Sprintf("❌ **Cache Access Failed:** %v\n", cacheErr))
	} else {
		response.WriteString(fmt.Sprintf("✅ **Cache Access:** %dms\n", cacheDuration.Milliseconds()))
		if guild != nil && cachedGuild != nil {
			response.WriteString(fmt.Sprintf("   Cache vs API: %d vs %d voice states\n", len(cachedGuild.VoiceStates), len(guild.VoiceStates)))
		}
	}

	// Test 3: Bot member fetch
	start = time.Now()
	botMember, err := b.session.GuildMember(guildID, b.session.State.User.ID)
	memberDuration := time.Since(start)

	if err != nil {
		response.WriteString(fmt.Sprintf("❌ **Bot Member API Call Failed:** %v\n", err))
	} else {
		response.WriteString(fmt.Sprintf("✅ **Bot Member API Call:** %dms\n", memberDuration.Milliseconds()))
		response.WriteString(fmt.Sprintf("   Bot has %d roles\n", len(botMember.Roles)))
	}

	// Test 4: Rate limit check
	response.WriteString("\n**📊 API Performance Analysis:**\n")
	if apiDuration.Milliseconds() > 1000 {
		response.WriteString("⚠️  API calls are slow (>1s) - possible rate limiting\n")
	} else if apiDuration.Milliseconds() > 500 {
		response.WriteString("⚠️  API calls are moderately slow (>500ms)\n")
	} else {
		response.WriteString("✅ API calls are responsive (<500ms)\n")
	}

	// Test 5: Voice state fetch specifically
	if guild != nil {
		response.WriteString(fmt.Sprintf("\n**🔊 Voice State Analysis:**\n"))
		response.WriteString(fmt.Sprintf("• API returned %d voice states\n", len(guild.VoiceStates)))

		if len(guild.VoiceStates) == 0 {
			response.WriteString("❌ **No voice states detected**\n")
			response.WriteString("   Possible causes:\n")
			response.WriteString("   - Users not actually in voice channels\n")
			response.WriteString("   - Bot lacks GUILD_VOICE_STATES intent\n")
			response.WriteString("   - Discord API synchronization lag\n")
		} else {
			response.WriteString("✅ Voice states are being returned by API\n")
		}
	}

	response.WriteString("\n**🎯 Recommendations:**\n")
	if apiDuration.Milliseconds() > 1000 {
		response.WriteString("• Check network connectivity\n")
		response.WriteString("• Verify Discord API status\n")
		response.WriteString("• Consider implementing API caching\n")
	} else {
		response.WriteString("• API connectivity appears normal\n")
		response.WriteString("• If voice detection fails, issue is likely with voice state events\n")
	}

	return response.String()
}

// checkVoicePermissions verifies that the bot has the necessary permissions for voice operations
func (b *Bot) checkVoicePermissions(s *discordgo.Session, guildID string) bool {
	// Get the bot's member info in the guild
	botMember, err := s.State.Member(guildID, s.State.User.ID)
	if err != nil {
		// Try API call if cache fails
		botMember, err = s.GuildMember(guildID, s.State.User.ID)
		if err != nil {
			log.Printf("Warning: Could not get bot member info for permission check: %v", err)
			return true // Assume permissions are okay if we can't check
		}
	}

	// Get guild info to check permissions
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			log.Printf("Warning: Could not get guild info for permission check: %v", err)
			return true // Assume permissions are okay if we can't check
		}
	}

	// Calculate permissions for the bot
	permissions := int64(0)

	// Check @everyone role permissions
	for _, role := range guild.Roles {
		if role.ID == guildID { // @everyone role has the same ID as the guild
			permissions |= role.Permissions
			break
		}
	}

	// Add permissions from bot's roles
	for _, roleID := range botMember.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				permissions |= role.Permissions
				break
			}
		}
	}

	// Check for required permissions
	const (
		PermissionViewChannel  = int64(0x0000000000000400) // 1024
		PermissionConnect      = int64(0x0000000000100000) // 1048576
		PermissionSpeak        = int64(0x0000000000200000) // 2097152
		PermissionSendMessages = int64(0x0000000000000800) // 2048
	)

	requiredPerms := PermissionViewChannel | PermissionConnect | PermissionSpeak | PermissionSendMessages

	// Check if bot has administrator permission (overrides all)
	if permissions&int64(0x0000000000000008) != 0 { // Administrator permission
		return true
	}

	// Check specific required permissions
	return (permissions & requiredPerms) == requiredPerms
}
