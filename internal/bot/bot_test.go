package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/doomhound188/soulhound/internal/config"
)

func TestBotCreation(t *testing.T) {
	// Test bot creation with a fake token
	cfg := &config.Config{
		DiscordToken:  "Bot.fake.token",
		YouTubeToken:  "",
		SpotifyToken:  "",
		DefaultPlayer: "yt",
	}

	bot, err := New(cfg)
	if err != nil {
		t.Fatalf("Expected bot creation to succeed, got error: %v", err)
	}

	if bot == nil {
		t.Fatal("Expected bot to be non-nil")
	}

	if bot.queue == nil {
		t.Error("Expected bot queue to be initialized")
	}

	if bot.voiceConn == nil {
		t.Error("Expected bot voice connections map to be initialized")
	}

	if bot.voiceStates == nil {
		t.Error("Expected bot voice states map to be initialized")
	}

	if bot.youtubePlayer == nil {
		t.Error("Expected YouTube provider to be initialized")
	}

	if bot.spotifyPlayer == nil {
		t.Error("Expected Spotify provider to be initialized")
	}
}

func TestVoiceStateTracking(t *testing.T) {
	cfg := &config.Config{
		DiscordToken:  "Bot.fake.token",
		YouTubeToken:  "",
		SpotifyToken:  "",
		DefaultPlayer: "yt",
	}

	bot, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test voice state update handler
	guildID := "test-guild-123"
	userID := "test-user-456"
	channelID := "test-channel-789"

	// Simulate a voice state update
	vsu := &discordgo.VoiceStateUpdate{
		VoiceState: &discordgo.VoiceState{
			UserID:    userID,
			ChannelID: channelID,
			GuildID:   guildID,
			SessionID: "test-session",
			Deaf:      false,
			Mute:      false,
			SelfDeaf:  false,
			SelfMute:  false,
			Suppress:  false,
		},
	}

	// Call the voice state update handler
	bot.voiceStateUpdateHandler(nil, vsu)

	// Check if voice state was tracked
	bot.mu.Lock()
	key := guildID + ":" + userID
	vs, exists := bot.voiceStates[key]
	bot.mu.Unlock()

	if !exists {
		t.Error("Expected voice state to be tracked after update")
	}

	if vs.ChannelID != channelID {
		t.Errorf("Expected channel ID %s, got %s", channelID, vs.ChannelID)
	}

	if vs.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, vs.UserID)
	}

	if vs.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, vs.GuildID)
	}

	// Test voice state removal
	vsuLeave := &discordgo.VoiceStateUpdate{
		VoiceState: &discordgo.VoiceState{
			UserID:    userID,
			ChannelID: "", // Empty channel ID means user left
			GuildID:   guildID,
		},
	}

	bot.voiceStateUpdateHandler(nil, vsuLeave)

	// Check if voice state was removed
	bot.mu.Lock()
	_, exists = bot.voiceStates[key]
	bot.mu.Unlock()

	if exists {
		t.Error("Expected voice state to be removed after user left")
	}
}

func TestHandleCommands(t *testing.T) {
	cfg := &config.Config{
		DiscordToken:  "Bot.fake.token",
		YouTubeToken:  "",
		SpotifyToken:  "",
		DefaultPlayer: "yt",
	}

	bot, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test help command
	response, err := bot.HandleCommand("help", []string{}, "", "", "")
	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}
	if response == "" {
		t.Error("Help command should return non-empty response")
	}

	// Test debug command
	response, err = bot.HandleCommand("debug", []string{}, "", "test-guild", "")
	if err != nil {
		t.Errorf("Debug command failed: %v", err)
	}
	if response == "" {
		t.Error("Debug command should return non-empty response")
	}

	// Test unknown command
	_, err = bot.HandleCommand("unknowncommand", []string{}, "", "", "")
	if err == nil {
		t.Error("Unknown command should return error")
	}
}

func TestStreamAudioMock(t *testing.T) {
	cfg := &config.Config{
		DiscordToken:  "Bot.fake.token",
		YouTubeToken:  "",
		SpotifyToken:  "",
		DefaultPlayer: "yt",
	}

	bot, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test with mock URL (should not fail for mock audio)
	vc := &VoiceConnection{
		connection: nil, // This will cause issues in real use but is ok for testing the URL detection logic
		channelID:  "test-channel",
		guildID:    "test-guild",
	}

	// Test mock URL detection
	err = bot.streamAudio("mock_test", vc)
	if err == nil {
		t.Error("Expected error when trying to stream with nil connection")
	}

	// Test YouTube URL detection
	err = bot.streamAudio("dQw4w9WgXcQ", vc)
	if err == nil {
		t.Error("Expected error explaining YouTube streaming requirements")
	}
	if err != nil && err.Error() != "YouTube audio streaming requires additional setup. Please install youtube-dl or yt-dlp for audio extraction" {
		t.Errorf("Expected specific YouTube error message, got: %v", err)
	}

	// Test Spotify ID detection
	err = bot.streamAudio("4iV5W9uYEdYUVa79Axb7Rh", vc)
	if err == nil {
		t.Error("Expected error explaining Spotify streaming limitations")
	}
	if err != nil && err.Error() != "Spotify audio streaming is not supported. Spotify does not provide direct audio streams" {
		t.Errorf("Expected specific Spotify error message, got: %v", err)
	}
}