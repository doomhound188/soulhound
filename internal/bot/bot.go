package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/doomhound/soulhound/internal/audio"
	"github.com/doomhound/soulhound/internal/config"
	"github.com/doomhound/soulhound/internal/queue"
	"github.com/jonas747/dca"
)

type Bot struct {
	session       *discordgo.Session
	queue         *queue.Queue
	youtubePlayer *audio.YouTubeProvider
	spotifyPlayer *audio.SpotifyProvider
	voiceConn     map[string]*VoiceConnection
	mu            sync.Mutex
	isPlaying     bool
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
	}

	session.AddHandler(bot.messageHandler)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

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

	// Get voice channel of the user
	voiceState, err := s.State.VoiceState(m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: You must be in a voice channel")
		return
	}

	response, err := b.HandleCommand(command, args, voiceState.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %s", err))
		return
	}

	s.ChannelMessageSend(m.ChannelID, response)
}

func (b *Bot) HandleCommand(command string, args []string, channelID string) (string, error) {
	switch strings.ToLower(command) {
	case "play":
		return b.handlePlay(args, channelID)
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
	default:
		return "", errors.New("unknown command")
	}
}

func (b *Bot) handlePlay(args []string, channelID string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("please provide a search query")
	}

	// Join voice channel first
	guildID := b.session.State.Ready.Guilds[0].ID // For simplicity, using first guild
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

	return fmt.Sprintf("Added to queue: %s - %s", track.Title, track.Artist), nil
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

		b.mu.Unlock()

		// Stream to all connected voice channels
		for guildID, vc := range b.voiceConn {
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
	conn, err := b.session.ChannelVoiceJoin(guildID, channelID, false, true)
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
	// Create DCA encoding session
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96

	encodingSession, err := dca.EncodeFile(url, options)
	if err != nil {
		return err
	}
	defer encodingSession.Cleanup()

	vc.encoder = encodingSession
	done := make(chan error)
	stream := dca.NewStream(encodingSession, vc.connection, done)
	vc.stream = stream

	err = <-done
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
			Platform: track.Platform,
			Genre:    result.Genre,
		})
	}
}
