package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
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
			Name:        "play",
			Description: "Play a youtube video in your current voice channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "url",
					Description: "YouTube URL",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"play": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			url := i.ApplicationCommandData().Options[0].StringValue()

			voiceChannelID := ""

			guild, err := s.State.Guild(i.GuildID)

			if err != nil {
				log.Fatal("couldn't retrieve guild from state object!")
			}

			for _, state := range guild.VoiceStates {
				if state.UserID == i.Member.User.ID {
					voiceChannelID = state.ChannelID
					break
				}
			}

			if voiceChannelID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You need to be in a voice channel to use this command.",
					},
				})
				return
			}

			vc, err := s.ChannelVoiceJoin(i.GuildID, voiceChannelID, false, true)
			if err != nil {
				log.Fatal("Error joining voice channel:", err)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Joining voice channel and playing: " + url,
				},
			})

			streamYouTubeAudio(vc, url)
		},
	}
)

func init() {
	s, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal("Invalid bot parameters: %v", err)
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

func streamYouTubeAudio(vc *discordgo.VoiceConnection, url string) {
	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		log.Fatal("Error retrieving YouTube video:", err)
	}

	// Find an audio-only format (based on MimeType)
	var audioFormat *youtube.Format
	for _, format := range video.Formats {
		// Check for audio-only formats (example: audio/opus or audio/webm)
		if format.MimeType == "audio/webm; codecs=\"opus\"" {
			audioFormat = &format
			break
		}
	}

	if audioFormat == nil {
		fmt.Println("No suitable audio format found")
		return
	}

	stream, _, err := client.GetStream(video, audioFormat)
	if err != nil {
		fmt.Println("Error getting video stream:", err)
		return
	}

	/* TODO: figure out how to play the stream */
}
