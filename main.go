package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"

	// youtube "github.com/UFeindschiff/youtube"
	youtube "github.com/MiguelCiulog/youtube-fork"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// Discord initialization
var (
	bot       *discordgo.Session
	prefix    = "-"
	serverIDs = []string{""} // add your server ID here
)

var botToken = "ODg3MzgzNDU4MDEwMjQ3MTY4.GA4KBm.NfK1Dm_O7qRtliGQTsPBPpH_dWhR1rWsIb-kKw"

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// args := os.Args
	// if len(args) < 2 {
	// 	log.Fatal("No token provided...")
	// }
	var err error
	bot, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Can't create session from the token: %v", err)
	}
}

// Youtube initialization
var (
	yt            youtube.Client
	encodeOptions *dca.EncodeOptions
)

func init() {
	yt = youtube.NewClient()
	encodeOptions = dca.StdEncodeOptions
	encodeOptions.RawOutput = true
	encodeOptions.Bitrate = 24
	encodeOptions.Application = "lowdelay"
}

func getStreamURL(videoName string) (string, error) {
	fmt.Println(videoName)

	results, err := youtube.Search(videoName, 0)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("Got %d search result(s).\n\n", len(results.Items))

	if len(results.Items) == 0 {
		if err != nil {
			log.Fatal("got zero search results")
		}
	}

	// Get the first search result and print out its details.

	details := results.Items[0]

	// fmt.Printf(
	// 	"ID: %q\n\nTitle: %q\nAuthor: %q\nDuration: %q\n\nView Count: %q\nLikes: %d\nDislikes: %d\n\n",
	// 	details.ID,
	// 	details.Title,
	// 	details.Author,
	// 	details.Duration,
	// 	details.Views,
	// 	details.Likes,
	// 	details.Dislikes,
	// )
	url_parsed := "https://www.youtube.com/watch?v=" + details.ID

	player, err := yt.Load(url_parsed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Title: %q\nAuthor: %q\nView Count: %s\n\n",
		player.Title(),
		player.Author(),
		player.ViewCount(),
	)

	stream, ok := player.SourceFormats().AudioOnly().BestAudio()
	if !ok {
		if err != nil {
			log.Fatal(err)
			return "", err
		}
	}

	url, err := player.ResolveURL(stream)
	fmt.Println(url)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	fmt.Println("err", err)

	return url, nil
}

func MessageResponseHandler(bot *discordgo.Session, m *discordgo.MessageCreate) {
	// Check for message
	// Ignore bot messages
	if m.Author.Bot || !strings.HasPrefix(m.Content, prefix) {
		return
	}

	message := strings.TrimPrefix(m.Content, prefix)
	urlMessage := strings.Split(message, " ")
	switch {
	case urlMessage[0] == "play" || urlMessage[0] == "p":
		channel, err := bot.State.Channel(m.ChannelID)
		if err != nil {
			return
		}
		guild, err := bot.State.Guild(channel.GuildID)
		if err != nil {
			return
		}

		channelID := ""
		for _, vs := range guild.VoiceStates {
			if vs.UserID == m.Author.ID {
				channelID = vs.ChannelID
				break
			}
		}

		if channelID == "" {
			bot.ChannelMessageSend(m.ChannelID, "You aren't in a voice channel")
			return
		}

		videoName := strings.Join(urlMessage[1:], " ")

		url, err := getStreamURL(videoName)
		if err != nil {
			bot.ChannelMessageSend(m.ChannelID, "line 98:")
			bot.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}

		encodingSession, err := dca.EncodeFile(url, encodeOptions)
		if err != nil {
			bot.ChannelMessageSend(m.ChannelID, "line 104:")
			bot.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}

		defer encodingSession.Cleanup()
		vc, err := bot.ChannelVoiceJoin(guild.ID, channelID, false, true)
		if err != nil {
			if _, ok := bot.VoiceConnections[guild.ID]; ok {
				vc = bot.VoiceConnections[guild.ID]
			} else {
				bot.ChannelMessageSend(m.ChannelID, err.Error())
				return
			}
		}
		vc.Speaking(true)
		done := make(chan error)
		dca.NewStream(encodingSession, vc, done)
		err = <-done
		if err != nil && err != io.EOF {
			bot.ChannelMessageSend(m.ChannelID, err.Error())
		}
		vc.Speaking(false)
		vc.Disconnect()
	}
}

func EmojiResponseHandler(bot *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Username == "Chaletlnwza007" {
		bot.MessageReactionAdd(m.ChannelID, m.ID, "😡")
		bot.MessageReactionAdd(m.ChannelID, m.ID, "🤢")
		return
	}
}

func init() {
	bot.AddHandler(MessageResponseHandler)
	bot.AddHandler(EmojiResponseHandler)
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "champ",
		Description: "cool pictures from Champ",
	},
	{
		Name:        "fluke",
		Description: "cool pictures from Fluke",
	},
	{
		Name:        "gift",
		Description: "receive a random superidol link",
	},
	{
		Name:        "lyrics",
		Description: "show superidol lyrics",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Name:        "language",
				Description: "choose the lyrics' language",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "chinese",
						Description: "Chinese ver.",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "pinyin",
						Description: "Chinese ver. but pinyin",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "romaji",
						Description: "Japanese ver. with romaji",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "thai",
						Description: "Thai ver.",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
				},
			},
		},
	},
}

func main() {
	bot.AddHandler(func(bot *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	err := bot.Open()
	if err != nil {
		log.Fatalf("Can't start session: %v", err)
	}
	defer bot.Close()

	for _, cmd := range commands {
		if len(serverIDs) == 0 {
			_, err = bot.ApplicationCommandCreate(bot.State.User.ID, "", cmd)
			if err != nil {
				log.Printf("Can't create a command: %v\n", err)
			}
		} else {
			for _, serverID := range serverIDs {
				_, err = bot.ApplicationCommandCreate(bot.State.User.ID, serverID, cmd)
				if err != nil {
					log.Printf("GuildID(%v) - Can't create a command: %v\n", serverID, err)
				}
			}
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Bot is down...")
}
