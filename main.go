package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

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
			ytbURL := i.ApplicationCommandData().Options[0].StringValue()

			// Join the user's voice channel
			guildID := i.GuildID
			voiceChannelID := getUserVoiceChannelID(s, guildID, i.Member.User.ID)

			if voiceChannelID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You must be in a voice channel to use this command.",
					},
				})
				return
			}

			// Join the voice channel
			vc, err := s.ChannelVoiceJoin(guildID, voiceChannelID, false, true)
			vc.OpusSend = make(chan []byte, 1024)
			if err != nil {
				log.Printf("error joining voice channel: %v", err)
				return
			}

			// Play the YouTube audio
			go playYouTubeAudio(ytbURL, vc)
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

func playYouTubeAudio(ytbURL string, vc *discordgo.VoiceConnection) {
	// Download YouTube audio using yt-dlp
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "-o", "-", ytbURL)
	audioStream, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("error creating stdout pipe: %v", err)
		return
	}
	defer audioStream.Close()

	err = cmd.Start()
	if err != nil {
		log.Printf("error starting yt-dlp command: %v", err)
		return
	}

	// Pass the YouTube stream through FFmpeg
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpeg.Stdin = audioStream
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Printf("error creating ffmpeg stdout pipe: %v", err)
		return
	}
	defer ffmpegOut.Close()

	err = ffmpeg.Start()
	if err != nil {
		log.Printf("error starting ffmpeg command: %v", err)
		return
	}

	// Send the audio to Discord
	sendPCMToDiscord(ffmpegOut, vc)
}

func sendPCMToDiscord(stream io.Reader, vc *discordgo.VoiceConnection) {
	// Create a buffer for 20ms of audio data (960 samples per frame for 48kHz stereo, 16-bit PCM)
	buffer := make([]byte, 1920) // 960 samples * 2 bytes per sample * 2 channels

	for {
		n, err := stream.Read(buffer)

		// If no data was read or an EOF is encountered, break the loop
		if err != nil {
			if err == io.EOF {
				log.Println("Audio stream ended.")
				break
			}
			log.Printf("Error reading from stream: %v", err)
			break
		}

		// If no data was returned, break the loop (prevent infinite loop on empty reads)
		if n == 0 {
			log.Println("No audio data read, stopping playback.")
			break
		}

		// Send the PCM data to Discord's OpusSend channel
		select {
		case vc.OpusSend <- buffer[:n]:
		default:
			log.Println("Voice connection buffer is full, dropping frame.")
		}

		// Add a delay to prevent flooding the OpusSend channel
		time.Sleep(20 * time.Millisecond) // 20ms per audio frame
	}

	log.Println("Finished sending audio to Discord.")
}

func getUserVoiceChannelID(s *discordgo.Session, guildID, userID string) string {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		log.Printf("error getting guild: %v", err)
		return ""
	}

	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}

	return ""
}
