# Max Bot

A simple bot implementation for Max Messenger using Go.

## Features

- Send and receive messages in Russian
- Basic command handling (hello, help, time, echo)
- Group chat support with message storage for future analysis
- Special admin commands for designated user (+79310071775)
- Proper handling of bot mentions in group chats
- Rate limiting to prevent API abuse
- Graceful shutdown handling
- Environment-based configuration

## Prerequisites

- Go 1.16 or later
- A Max Messenger bot token
- GitHub CLI (for synchronization with GitHub)

## Setup

1. Clone or download this repository
2. Install dependencies:

```bash
go mod tidy
```

3. Set up your bot token either as an environment variable or in a .env file:

Option 1: Environment variable
```bash
export MAX_BOT_TOKEN="your_bot_token_here"
```

On Windows:
```cmd
set MAX_BOT_TOKEN=your_bot_token_here
```

Option 2: Create a .env file in the project root with your token:
```
MAX_BOT_TOKEN=your_bot_token_here
```

Note: A .env file template is provided as .env.example that you can copy and modify.

## How to Run

1. Make sure you have set the `MAX_BOT_TOKEN` either as an environment variable or in a .env file
2. Run the bot:

```bash
go run main.go
```

The bot will start and begin listening for messages.

## Configuration

The bot uses the following environment variables:

- `MAX_BOT_TOKEN`: Your Max Messenger bot token (required)

## Supported Commands

### For all users (in private messages):
- `привет`, `здравствуй`, `добрый день`, `hello`, `hi`, `hey`: Greeting response
- `помощь`, `help`: Help information
- `время`, `time`: Current timestamp
- `повтори [text]`, `скажи [text]`, `echo [text]`: Echoes back the provided text

### For admin user (+79310071775) (in private messages):
- `привет`, `здравствуй`, `добрый день`, `hello`, `hi`, `hey`: Admin greeting
- `помощь`, `help`: Admin commands list
- `статистика`, `stats`: Total message count in storage
- `история`, `history`: Last 5 messages from storage

### Group chat behavior:
- When bot is mentioned (with "@bot", "бот," "бот " etc.), responds with "Извините, я пока не умею разговаривать."
- All messages in group chats are stored for future analysis
- Bot does not respond to other messages in group chats

## Project Structure

```
max-bot/
├── bot/
│   └── client.go       # Bot API client implementation
├── handlers/
│   └── handler.go      # Message handling logic with message storage
├── utils/
│   └── utils.go        # Utility functions
├── main.go             # Main application entry point
├── example.go          # Example usage
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
└── README.md           # This file
```

## Extending the Bot

To add new functionality:

1. Modify `handlers/handler.go` to add new command handlers
2. Update the `Handle` function to recognize new commands
3. Add any necessary helper functions

## Synchronization with GitHub

This project supports synchronization with GitHub using GitHub CLI:

1. Make sure you have GitHub CLI installed and authenticated
2. Commit your changes:
```bash
git add .
git commit -m "Your commit message"
```
3. Push to GitHub:
```bash
gh repo sync
```
Or alternatively:
```bash
git push origin main
```

## License

This project is open source and available under the MIT License.