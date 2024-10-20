package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")

	if err != nil {
	  log.Fatalf("Error loading .env file")
	}
  
	discord, err := discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))

	if err != nil {
		log.Fatal("couldn't create session!")
		return
	}

	discord.AddHandler(messageHandler)

	err = discord.Open()

	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	fmt.Println("Bot is running. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)
    <-sc

	discord.Close()
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println("Message received: ", m.Content)

	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}
