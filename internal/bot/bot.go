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
}

func (b *Bot) voiceStateUpdateHandler(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	// Update our internal state tracking
	// This handler ensures we have the most up-to-date voice state information
	if vsu.ChannelID == "" {
		log.Printf("User %s left voice channel in guild %s", vsu.UserID, vsu.GuildID)
	} else {
		log.Printf("User %s joined voice channel %s in guild %s", vsu.UserID, vsu.ChannelID, vsu.GuildID)
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

		// Try multiple methods to get voice state
		var voiceState *discordgo.VoiceState
		log.Printf("Voice detection: Starting voice state lookup for user %s (%s) in guild %s", m.Author.Username, m.Author.ID, m.GuildID)

		// Method 1: Try to get voice state from cache
		voiceState, err := s.State.VoiceState(m.GuildID, m.Author.ID)
		if err != nil {
			log.Printf("Voice detection: Cache lookup failed: %v", err)
		} else if voiceState != nil && voiceState.ChannelID != "" {
			log.Printf("Voice detection: Found voice state in cache - Channel: %s", voiceState.ChannelID)
		} else {
			log.Printf("Voice detection: Cache lookup returned nil or empty channel")
		}

		if err != nil || voiceState == nil || voiceState.ChannelID == "" {
			// Method 2: Search through all cached voice states in the guild
			log.Printf("Voice detection: Trying method 2 - searching guild voice states")
			guild, err2 := s.State.Guild(m.GuildID)
			if err2 == nil && guild != nil {
				log.Printf("Voice detection: Found guild with %d voice states", len(guild.VoiceStates))
				for _, vs := range guild.VoiceStates {
					if vs.UserID == m.Author.ID && vs.ChannelID != "" {
						log.Printf("Voice detection: Found matching voice state in guild cache - Channel: %s", vs.ChannelID)
						voiceState = vs
						break
					}
				}
			} else {
				log.Printf("Voice detection: Could not get guild from cache: %v", err2)
			}
		}

		// Method 3: If still no voice state, make a direct API call
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 3 - direct API call")
			// Get fresh guild data from Discord API
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

		// Method 4: Last resort - wait a moment and try cache again
		if voiceState == nil || voiceState.ChannelID == "" {
			log.Printf("Voice detection: Trying method 4 - retry after delay")
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

	response, err := b.HandleCommand(command, args, voiceChannelID, m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %s", err))
		return
	}

	if response != "" {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}

func (b *Bot) HandleCommand(command string, args []string, channelID string, guildID string) (string, error) {
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
	// Validate URL
	if url == "" {
		return fmt.Errorf("empty stream URL")
	}

	// Create DCA encoding session
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96

	// For URLs that are not directly streamable (like Spotify URLs or YouTube URLs without extraction),
	// we should ideally use youtube-dl or similar tools. For now, we'll handle the error gracefully.
	encodingSession, err := dca.EncodeFile(url, options)
	if err != nil {
		// Log the error but don't fail completely
		log.Printf("Warning: Could not encode audio from URL %s: %v", url, err)
		return fmt.Errorf("audio streaming not available for this source: %w", err)
	}
	defer encodingSession.Cleanup()

	vc.encoder = encodingSession
	done := make(chan error)
	stream := dca.NewStream(encodingSession, vc.connection, done)
	vc.stream = stream

	err = <-done
	if err != nil {
		log.Printf("Streaming finished with error: %v", err)
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

	// Check bot permissions
	debugInfo.WriteString(fmt.Sprintf("**Bot Intents:** %d\n", b.session.Identify.Intents))
	debugInfo.WriteString("**Required Intents:** GuildMessages + GuildVoiceStates + MessageContent\n\n")

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

	// Get the bot's member info in the guild
	botMember, err := b.session.State.Member(guildID, b.session.State.User.ID)
	if err != nil {
		botMember, err = b.session.GuildMember(guildID, b.session.State.User.ID)
		if err != nil {
			return fmt.Sprintf("❌ Could not get bot member info: %v\n", err)
		}
	}

	// Get guild info
	guild, err := b.session.State.Guild(guildID)
	if err != nil {
		guild, err = b.session.Guild(guildID)
		if err != nil {
			return fmt.Sprintf("❌ Could not get guild info: %v\n", err)
		}
	}

	// Calculate total permissions
	permissions := int64(0)

	// Check @everyone role permissions
	for _, role := range guild.Roles {
		if role.ID == guildID { // @everyone role
			permissions |= role.Permissions
			permInfo.WriteString(fmt.Sprintf("@everyone permissions: %d\n", role.Permissions))
			break
		}
	}

	// Add permissions from bot's roles
	permInfo.WriteString("Bot roles:\n")
	for _, roleID := range botMember.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				permissions |= role.Permissions
				permInfo.WriteString(fmt.Sprintf("• %s: %d\n", role.Name, role.Permissions))
				break
			}
		}
	}

	permInfo.WriteString(fmt.Sprintf("**Total calculated permissions: %d**\n", permissions))

	// Check specific permissions
	const (
		PermissionViewChannel  = int64(1024)    // 0x400
		PermissionConnect      = int64(1048576) // 0x100000
		PermissionSpeak        = int64(2097152) // 0x200000
		PermissionSendMessages = int64(2048)    // 0x800
	)

	permInfo.WriteString("**Required Permission Check:**\n")

	if permissions&int64(8) != 0 { // Administrator
		permInfo.WriteString("✅ Administrator (has all permissions)\n")
	} else {
		if permissions&PermissionViewChannel != 0 {
			permInfo.WriteString("✅ View Channels\n")
		} else {
			permInfo.WriteString("❌ View Channels\n")
		}

		if permissions&PermissionSendMessages != 0 {
			permInfo.WriteString("✅ Send Messages\n")
		} else {
			permInfo.WriteString("❌ Send Messages\n")
		}

		if permissions&PermissionConnect != 0 {
			permInfo.WriteString("✅ Connect\n")
		} else {
			permInfo.WriteString("❌ Connect\n")
		}

		if permissions&PermissionSpeak != 0 {
			permInfo.WriteString("✅ Speak\n")
		} else {
			permInfo.WriteString("❌ Speak\n")
		}
	}

	return permInfo.String()
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
