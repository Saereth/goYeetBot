package main

import (
        "encoding/csv"
        "fmt"
        "log"
        "os"
        "time"

        "gopkg.in/yaml.v3"

        "github.com/bwmarrin/discordgo"
)

type Config struct {
        Token          string `yaml:"token"`
        GuildID        string `yaml:"guild_id"`
        CSVOutput      string `yaml:"csv_output"`
        InactivityDays int    `yaml:"inactivity_days"`
        Debug          bool   `yaml:"debug"`
}

func main() {
	// Load configuration from YAML file
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
		return
	}

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
		return
	}
	defer dg.Close()

	fmt.Println("Bot is now running. Fetching guild information...")

	// Get the guild (server) by ID
	guild, err := dg.Guild(config.GuildID)
	if err != nil {
		log.Fatalf("Error fetching guild: %v", err)
		return
	}

	fmt.Printf("Connected to guild: %s (%s)\n", guild.Name, config.GuildID)

	// Fetch the channels directly using the Discord API
	channels, err := dg.GuildChannels(config.GuildID)
	if err != nil {
		log.Fatalf("Error fetching channels: %v", err)
		return
	}

	// Time thresholds
	inactiveThreshold := time.Now().AddDate(0, 0, -config.InactivityDays)
	historyLimit := inactiveThreshold

	fmt.Printf("Inactivity threshold: %s\n", inactiveThreshold)
	fmt.Printf("History limit: %s\n", historyLimit)

	// Store the last message time for each user
	lastMessageTimes := make(map[string]time.Time)

	// Iterate over all channels and process only text channels
	for _, channel := range channels {
		if channel.Type != discordgo.ChannelTypeGuildText {
			continue
		}

		fmt.Printf("Processing text channel: %s\n", channel.Name)

		// Fetch and process messages in the channel
		messages, err := fetchChannelMessages(dg, channel.ID, historyLimit, config.Debug)
		if err != nil {
			log.Printf("Could not access channel %s: %v", channel.Name, err)
			continue
		}

		processMessages(messages, lastMessageTimes, config.Debug, channel.Name)

		// Fetch and process active threads
		activeThreads, err := dg.ThreadsActive(channel.ID)
		if err == nil {
			for _, thread := range activeThreads.Threads {
				fmt.Printf("Processing active thread: %s\n", thread.Name)
				messages, err := fetchChannelMessages(dg, thread.ID, historyLimit, config.Debug)
				if err != nil {
					log.Printf("Could not access thread %s: %v", thread.Name, err)
					continue
				}
				processMessages(messages, lastMessageTimes, config.Debug, thread.Name)
			}
		}

		// Fetch and process archived threads
		limit := 50
		archivedThreads, err := dg.ThreadsArchived(channel.ID, &historyLimit, limit)
		if err == nil {
			for _, thread := range archivedThreads.Threads {
				fmt.Printf("Processing archived thread: %s\n", thread.Name)
				messages, err := fetchChannelMessages(dg, thread.ID, historyLimit, config.Debug)
				if err != nil {
					log.Printf("Could not access thread %s: %v", thread.Name, err)
					continue
				}
				processMessages(messages, lastMessageTimes, config.Debug, thread.Name)
			}
		}
	}

	// Prepare CSV output
	file, err := os.Create(config.CSVOutput)
	if err != nil {
		log.Fatalf("Could not create CSV file: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Username", "Last Message Time"})

	fmt.Println("Evaluating each member's activity...")

	// Evaluate each member's activity
	for _, member := range guild.Members {
		if member.User.Bot {
			if config.Debug {
				fmt.Printf("Skipping bot user: %s\n", member.User.Username)
			}
			continue // Skip bots
		}

		lastMessageTime, exists := lastMessageTimes[member.User.ID]
		if !exists {
			// User has never sent a message
			writer.Write([]string{member.User.Username, "Never sent a message"})
			fmt.Printf("User %s has never sent a message, added to list as inactive.\n", member.User.Username)
		} else if lastMessageTime.Before(inactiveThreshold) {
			// User has been inactive
			writer.Write([]string{member.User.Username, lastMessageTime.Format(time.RFC3339)})
			fmt.Printf("User %s has been inactive since %s, added to list.\n", member.User.Username, lastMessageTime)
		} else if config.Debug {
			fmt.Printf("User %s is active, last message on %s.\n", member.User.Username, lastMessageTime)
		}
	}

	fmt.Printf("Saved inactive members to %s\n", config.CSVOutput)
}

// fetchChannelMessages fetches messages from a channel after a specific timestamp.
func fetchChannelMessages(s *discordgo.Session, channelID string, after time.Time, debug bool) ([]*discordgo.Message, error) {
	var allMessages []*discordgo.Message
	var lastMessageID string

	for {
		if debug {
			fmt.Printf("Fetching messages before ID %s\n", lastMessageID)
		}
		messages, err := s.ChannelMessages(channelID, 100, "", lastMessageID, "")
		if err != nil {
			return nil, err
		}
		if len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			if debug {
				// Output each message as it is fetched
				fmt.Printf("Fetched message: Author=%s, Timestamp=%s, Content=%s\n", msg.Author.Username, msg.Timestamp, msg.Content)
			}

			if msg.Timestamp.Before(after) {
				if debug {
					fmt.Printf("Message is before history limit (%s), stopping fetch.\n", after)
				}
				return allMessages, nil
			}
			allMessages = append(allMessages, msg)
		}

		lastMessageID = messages[len(messages)-1].ID
		if debug {
			fmt.Printf("Fetched %d messages, last message ID: %s\n", len(messages), lastMessageID)
		}
	}

	return allMessages, nil
}

// processMessages processes messages and updates the last message time for users
func processMessages(messages []*discordgo.Message, lastMessageTimes map[string]time.Time, debug bool, location string) {
	totalMessages := len(messages)
	for i, msg := range messages {
		if debug {
			fmt.Printf("Reviewing message in %s: Author=%s, Timestamp=%s, Content=%s\n", location, msg.Author.Username, msg.Timestamp, msg.Content)
		}

		// Skip messages from bots
		if msg.Author.Bot {
			continue
		}

		// Update the last message time if it's more recent
		if lastMessage, exists := lastMessageTimes[msg.Author.ID]; !exists || msg.Timestamp.After(lastMessage) {
			lastMessageTimes[msg.Author.ID] = msg.Timestamp
			if debug {
				fmt.Printf("Updated last message time for user %s: %s\n", msg.Author.Username, msg.Timestamp)
			}
		}

		// Calculate and output the percentage of messages reviewed
		if totalMessages > 0 && (i+1)%10 == 0 { // Adjust this condition to change how often the percentage is output
			percentComplete := float64(i+1) / float64(totalMessages) * 100
			fmt.Printf("Processed %.2f%% of messages in %s\n", percentComplete, location)
		}
	}
}

// loadConfig reads a YAML file and unmarshals it into a Config struct
func loadConfig(file string) (*Config, error) {
	config := &Config{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}