package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/AbdelazizTina-dev/go-discord-bot/scraper"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

var (
	s        *discordgo.Session
	err      error
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "kayn-update",
			Description: "Will tell you if there is an update in valo :)",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"kayn-update": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Acknowledge the interaction immediately, telling Discord the bot is working on it.
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			if err != nil {
				fmt.Println("Error acknowledging interaction:", err)
				return
			}

			patch := scraper.GetPatchNotes()

			embed := &discordgo.MessageEmbed{
				Title:       "VALORANT Patch Notes",
				Description: "Check out the latest patch notes for VALORANT!",
				Color:       0x00ff00, // Green color (in hexadecimal)
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Patch Version",
						Value:  patch.Version, // Replace with dynamic version number
						Inline: true,
					},
					{
						Name:   "Release Date",
						Value:  patch.Date, // Replace with dynamic date
						Inline: true,
					},
					{
						Name:   "Patch Description",
						Value:  patch.Description, // Summarized details
						Inline: false,
					},
				},
				URL: patch.Link, // Link to full patch notes
			}

			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    nil, // Content can be nil if you're only sending an embed
				Embeds:     &[]*discordgo.MessageEmbed{embed},
				Components: nil, // No components are needed in this case
			})

			if err != nil {
				fmt.Println("Error sending updated embed:", err)
			}

		}}
)

func init() {
	s, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	scraper.GetPatchNotes()
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err := s.Open()

	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, os.Getenv("GUILD_ID"), v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	log.Println("Gracefully shutting down.")
}
