package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/doomhound/soulhound/internal/bot"
	"github.com/doomhound/soulhound/internal/config"
)

func main() {
	// Command line flags
	discordToken := flag.String("discord", os.Getenv("DISCORD_TOKEN"), "Discord Bot Token")
	youtubeToken := flag.String("youtube", os.Getenv("YOUTUBE_TOKEN"), "YouTube API Token")
	spotifyToken := flag.String("spotify", os.Getenv("SPOTIFY_TOKEN"), "Spotify API Token")
	flag.Parse()

	// Check for Discord token in environment if not provided via flag
	if *discordToken == "" {
		log.Fatal("Discord token is required. Set it via -discord flag or DISCORD_TOKEN environment variable")
	}

	// Initialize configuration
	config.Init(*discordToken, *youtubeToken, *spotifyToken)

	// Create and start the bot
	discordBot, err := bot.New(&config.AppConfig)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	if err := discordBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// Set up signal handling for graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down...")
	if err := discordBot.Close(); err != nil {
		log.Printf("Error while shutting down: %v", err)
	}
}
