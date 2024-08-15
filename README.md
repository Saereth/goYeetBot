# Discord Inactivity Bot

This bot identifies inactive members in a Discord server based on their message activity. It outputs a CSV file listing members who haven't sent a message in a specified number of days.

## Features

- Fetches all text channels in the server and reviews message activity.
- Outputs a CSV file listing inactive members.
- Configurable through a YAML file.
- Optionally displays verbose debugging information.
- Provides real-time percentage feedback on message processing.

## Requirements

- [Go](https://golang.org/dl/) (1.16 or higher)
- A Discord bot token with appropriate permissions:
  - `View Channels`
  - `Read Message History`

## Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/yourusername/discord-inactivity-bot.git
   cd discord-inactivity-bot
   
2. **Install dependencies:**

   Make sure to install the `yaml.v3` package:

   ```bash
   go get gopkg.in/yaml.v3

## Configuration

Create a `config.yaml` file in the root of the project with the following structure:

```yaml
token: "your_discord_bot_token_here"
guild_id: "your_guild_id_here"
csv_output: "inactive_members.csv"
inactivity_days: 60
debug: false
```

## Running the Bot

1. Ensure your `config.yaml` file is properly configured.
2. Run the compiled program:

   ```bash
   ./discord-inactivity-bot
   ```

## Example Output

The bot will create a CSV file (by default `inactive_members.csv`) with the following structure:

| Username      | Last Message Time        |
| ------------- | ------------------------ |
| InactiveUser1 | Never sent a message     |
| InactiveUser2 | 2024-06-01T15:22:46.035Z |


## Notes
Ensure the bot has the necessary permissions in the Discord server to view channels and read message history.
The debug option in config.yaml can be set to true to enable detailed logging, which can help in debugging or understanding the bot's behavior.
Contributing Feel free to submit issues or pull requests if you have suggestions or improvements.

## License
This project is licensed under the MIT License.
