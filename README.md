# NCU CSIE Announcement Monitor

A Go application that monitors the NCU CSIE recruitment announcements page and sends notifications to Telegram and Discord when new announcements are posted.

## Features

- Monitors announcements every 1 minute
- Sends notifications to Telegram
- Sends notifications to Discord
- Persistent state tracking
- Automatic retry on errors

## Prerequisites

- Go 1.21 or higher
- A Telegram Bot (optional)
- A Discord Webhook (optional)

## Setup

### 1. Clone the repository

```bash
git clone https://github.com/yorukot/csie.ncu.git
cd csie.ncu
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure notifications

#### Telegram Setup

1. Create a bot with [@BotFather](https://t.me/BotFather)
   - Send `/newbot` and follow the instructions
   - Copy the bot token

2. Get your Chat ID
   - Send a message to [@userinfobot](https://t.me/userinfobot)
   - Copy your Chat ID

#### Discord Setup

1. Go to your Discord server settings
2. Navigate to Integrations > Webhooks
3. Create a new webhook
4. Copy the webhook URL

### 4. Set environment variables

Create a `.env` file or set environment variables:

```bash
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export TELEGRAM_CHAT_ID="your_chat_id_here"
export DISCORD_WEBHOOK_URL="your_discord_webhook_url_here"
```

Or copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
# Edit .env with your credentials
```

## Usage

### Run directly

```bash
TELEGRAM_BOT_TOKEN="your_token" TELEGRAM_CHAT_ID="your_chat_id" go run main.go
```

### Build and run

```bash
go build -o announcements-monitor
./announcements-monitor
```

### Run in background (Linux)

```bash
nohup ./announcements-monitor > monitor.log 2>&1 &
```

### Stop the monitor

Press `Ctrl+C` or kill the process:

```bash
pkill -f announcements-monitor
```

## Running on Linux Server

### Option 1: Using nohup

```bash
nohup TELEGRAM_BOT_TOKEN="your_token" TELEGRAM_CHAT_ID="your_chat_id" ./announcements-monitor > monitor.log 2>&1 &
```

### Option 2: Using systemd (recommended)

Create a service file `/etc/systemd/system/ncu-announcements.service`:

```ini
[Unit]
Description=NCU CSIE Announcement Monitor
After=network.target

[Service]
Type=simple
User=your_username
WorkingDirectory=/path/to/csie.ncu
Environment="TELEGRAM_BOT_TOKEN=your_token"
Environment="TELEGRAM_CHAT_ID=your_chat_id"
Environment="DISCORD_WEBHOOK_URL=your_webhook"
ExecStart=/path/to/csie.ncu/announcements-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ncu-announcements
sudo systemctl start ncu-announcements
sudo systemctl status ncu-announcements
```

## Configuration

- `checkInterval`: Check interval (default: 1 minute)
- `stateFile`: File to store last seen announcements (default: `last_announcements.json`)

## Output

The monitor will display:
- `✓` for regular checks with no new announcements
- `🔔` when new announcements are detected
- `✓ Telegram notification sent` when Telegram message is sent successfully
- `✓ Discord notification sent` when Discord message is sent successfully

## License

MIT License

## Contributing

Pull requests are welcome!
