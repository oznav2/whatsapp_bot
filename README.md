# WhatsApp_bot_multiuser

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.24.1-00ADD8?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)
![WhatsApp](https://img.shields.io/badge/WhatsApp-Business_API-25D366?style=for-the-badge&logo=whatsapp)

A powerful WhatsApp bot that provides real-time translation, media downloading, and entertainment features using advanced AI capabilities.

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Commands](#commands) • [Development](#development) • [Contributing](#contributing)

</div>

## 🌟 Features

### Core Capabilities
- **🌐 Real-time Translation**: Translate messages between 20+ languages using Google's Gemini AI
- **🎤 Auto Audio & Video Transcription**: Enable automatic transcription with `/tenable` for voice notes, video messages, and URLs
- **🔗 Smart URL Detection**: When `/tenable` is active, automatically transcribe any video URL sent in chat
- **🎬 Manual Video Transcription**: Transcribe videos from YouTube, Instagram, Twitter, TikTok, and 100+ platforms on demand
- **🎯 Multi-Model Support**: Hebrew-optimized Ivrit CT2, multilingual Whisper V3 Turbo, and cloud-based Deepgram Nova-3
- **📥 Media Downloader**: Download videos and images from YouTube, Instagram, Twitter, and more
- **🤖 AI-Powered**: Leverages Google Gemini 2.0 for translation and image generation
- **🎮 Entertainment**: Fun commands including memes, random emojis, and more
- **🔧 Extensible Architecture**: Easy-to-extend command framework for adding new features

### Technical Highlights
- Built with Go for high performance and reliability
- WhatsApp Multi-Device support via [whatsmeow](https://github.com/tulir/whatsmeow)
- Docker-ready for easy deployment
- Comprehensive command framework with middleware support
- Rate limiting and permission controls

## 📋 Prerequisites

- Go 1.24.1 or higher
- Docker and Docker Compose (optional)
- Google Cloud API key (for Gemini AI)
- WhatsApp account for bot usage

## 🚀 Installation

### Using Docker (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/ASparkOfFire/whatsapp-livetranslate-2.0.git
cd whatsapp-livetranslate-2.0
```

2. Create a `.env` file:
```env
GEMINI_KEY=your_gemini_api_key_here
COOKIES_PATH=/path/to/cookies.txt  # Optional: for social media downloads
```

3. Run with Docker Compose:
```bash
docker-compose up -d
```

### Manual Installation

1. Clone the repository:
```bash
git clone https://github.com/ASparkOfFire/whatsapp-livetranslate-2.0.git
cd whatsapp-livetranslate-2.0
```

2. Install dependencies:
```bash
go mod download
```

3. Install yt-dlp (for media downloading):
```bash
# macOS
brew install yt-dlp

# Ubuntu/Debian
sudo apt install yt-dlp

# Using pip
pip install yt-dlp
```

4. Create a `.env` file with your configuration

5. Build and run:
```bash
go build -o whatsapp-livetranslate .
./whatsapp-livetranslate
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GEMINI_KEY` | Google Gemini API key for AI features | Yes |
| `YOUTUBE_VISITOR_DATA` | YouTube visitor data for bypassing some restrictions | No |
| `COOKIES_PATH` | Path to cookies.txt for non-YouTube sites (Instagram, Twitter, etc.) | No |
| `HIBP_TOKEN` | API token for Have I Been Pwned dark web search (owner only) | No |
| `TRANSCRIBE_SERVICE_URL` | URL for audio transcription service (defaults to http://localhost:8009) | No |

### YouTube Visitor Data (Optional)

To bypass some YouTube restrictions without using cookies:

1. Open YouTube in your browser
2. Open Developer Tools (F12)
3. Go to Network tab
4. Play any video
5. Look for requests to `youtubei/v1/player` 
6. In the request headers or payload, find `visitorData` value
7. Set it as environment variable:
   ```bash
   export YOUTUBE_VISITOR_DATA="your_visitor_data_here"
   ```

This helps with:
- Some geo-restricted content
- Certain rate limits
- Age-restricted videos (limited effectiveness)

**Note**: This method is less reliable than cookies but doesn't require authentication.

### Cookies for Other Sites (Optional)

For non-YouTube sites (Instagram, Twitter, TikTok, etc.), you can use cookies:

1. Export cookies from your browser using a cookies extension
2. Save as `cookies.txt` in Netscape format
3. Set the environment variable:
   ```bash
   export COOKIES_PATH=/path/to/cookies.txt
   ```

This helps with:
- Private Instagram posts/stories
- Protected Twitter/X content
- TikTok private videos
- Other sites requiring authentication

**Note**: YouTube will always use visitor data instead of cookies for better stability.

### First-Time Setup

1. On first run, scan the QR code with WhatsApp to link your device
2. The bot will automatically save the session for future use
3. Send `/help` to see available commands

## 📱 Commands

### Translation Commands
- `/[language_code] <text>` - Translate text to specified language
- `/[language_code]` - Translate quoted message
- Examples: `/ar Hello world`, `/fr`, `/ja`

### Utility Commands
- `/help` - Show all available commands
- `/ping` - Check if bot is responsive
- `/supportedlangs` - List all supported languages
- `/download <url>` - Download media from social platforms
- `/dl <url>` - Alias for download
- `/transcribe <url>` - Transcribe video from URL (YouTube, Instagram, TikTok, Twitter, etc.)
- `/hibp <phone_or_identifier>` - Check if a phone or identifier has been exposed in data breaches, focusing on HiTeckGroop.in (owner only) - [Documentation](docs/HIBP_COMMAND.md)

### Fun Commands
- `/meme [subreddit]` - Get random meme (default: dankmemes)
- `/image <prompt>` - Generate AI image
- `/randmoji` - Spam random emojis
- `/haha [intensity]` - Generate laughter

### Admin Commands
- `/setmodel <model>` - Change AI model
- `/getmodel` - Show current AI model
- `/settemp <value>` - Set AI temperature
- `/gettemp` - Show current temperature

### Transcription Commands

#### Manual Transcription (Commands)
- `/transcribe <url>` - Transcribe Hebrew video using Ivrit CT2 (Hebrew-optimized local model)
- `/transcriben <url>` - Transcribe English/multilingual video using Whisper V3 Turbo (local model)
- `/transcribedg <url>` - Transcribe video using Deepgram Nova-3 (cloud-based API with auto language detection)

#### Automatic Transcription
- `/tenable [language]` - Enable automatic transcription for this chat (default: Hebrew)
- `/tdisable` - Disable automatic transcription for this chat

When `/tenable` is active, the bot automatically transcribes:
- **🎤 Audio messages**: Voice notes and audio files
- **🎬 Video messages**: Video files sent directly in chat
- **🔗 URLs in text messages**: YouTube, Instagram, Twitter, TikTok, and 100+ platforms

**Language Routing**:
- `/tenable` (or `/tenable he`) → Uses **Ivrit CT2** for Hebrew content
- `/tenable en` → Uses **Whisper V3 Turbo** for English/multilingual content
- Change language anytime by running `/tenable` again with different language code

**Per-Chat Settings**: Each chat has its own transcription settings (enable/disable independently)

#### Transcription Features

**Platform Support**:
- **🌐 100+ Platforms**: YouTube, Instagram, Twitter/X, TikTok, Facebook, Reddit, and more via yt-dlp
- **🎯 Smart Model Selection**:
  - **Ivrit CT2** - Hebrew-optimized local model (ivrit-ai/whisper-large-v3-turbo-ct2)
  - **Whisper V3 Turbo** - Fast multilingual local model
  - **Deepgram Nova-3** - Cloud-based API with auto language detection
- **📊 Video Metadata**: Automatically extracts title, duration, and thumbnail
- **⚡ Progress Tracking**: Real-time download and transcription progress updates
- **✅ Complete Results**: Final message includes video title, duration, detected language, and full transcription

**Example: Automatic URL Transcription**:
```
User: /tenable                    # Enable Hebrew transcription
Bot: ✅ Transcription enabled

User: https://youtube.com/watch?v=xyz
Bot: 🔗 מוריד מידע על הווידאו...
     📊 מקבל מידע על הווידאו...
     🎬 מתמלל בעברית...
     🎬 מתמלל וידאו... (הורדה: 25%)
     🎬 *תמלול הושלם*

     ⏱️ משך: 5min and 23 seconds
     🔤 שפה: Hebrew

     📝 *תמלול:*
     [Complete Hebrew transcription]
```

**Example: Manual Command Transcription**:
```
User: /transcribe https://youtube.com/watch?v=xyz
Bot: [YouTube preview link]
Bot: 🎬 *תמלול הושלם* (Ivrit CT2)

     📹 סרטון: "Video Title Here"
     ⏱️ משך: 5min and 23 seconds

     📝 *תמלול:*
     [Complete transcription]
```

## 🏗️ Architecture

### Project Structure
```
whatsapp-livetranslate-2.0/
├── cmd/                    # Application entry points
├── internal/              
│   ├── cmdframework/      # Command framework infrastructure
│   ├── handlers/          # Command implementations
│   │   ├── admin/        # Administrative commands
│   │   ├── fun/          # Entertainment commands
│   │   ├── translation/  # Translation commands
│   │   └── utility/      # Utility commands
│   ├── services/         # Core services
│   └── constants/        # Application constants
├── docs/                  # Documentation
├── docker-compose.yml    # Docker configuration
└── Dockerfile           # Container definition
```

### Command Framework

The bot uses a powerful command framework that makes it easy to add new commands:

```go
type Command interface {
    Execute(ctx *Context) error
    Metadata() *Metadata
}
```

See [docs/ADDING_COMMANDS.md](docs/ADDING_COMMANDS.md) for detailed documentation on creating new commands.

## 🔒 Security Features

- **Owner-only commands**: Sensitive commands restricted to bot owner
- **Rate limiting**: Prevents abuse with configurable limits
- **Download restrictions**: 1-minute cooldown between downloads
- **Input validation**: All user inputs are validated and sanitized

## 🛠️ Development

### Adding New Commands

1. Create a new file in the appropriate handler directory
2. Implement the `Command` interface
3. Register the command in `event_handler.go`

Example:
```go
type MyCommand struct{}

func (c *MyCommand) Execute(ctx *framework.Context) error {
    return ctx.Handler.SendResponse(
        ctx.MessageInfo, 
        "Hello from my command!",
    )
}

func (c *MyCommand) Metadata() *framework.Metadata {
    return &framework.Metadata{
        Name:        "mycommand",
        Description: "My custom command",
        Category:    "Custom",
    }
}
```

### Running Tests
```bash
go test ./...
```

### Building
```bash
go build -o whatsapp-livetranslate .
```

## 📊 Performance

- Handles multiple concurrent users
- Message processing < 100ms (excluding AI calls)
- Minimal memory footprint (~50MB base)
- Automatic reconnection on network issues

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new features
- Update documentation as needed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [whatsmeow](https://github.com/tulir/whatsmeow) - WhatsApp Web API
- [go-ytdlp](https://github.com/lrstanley/go-ytdlp) - YouTube-DL wrapper
- [Google Gemini](https://ai.google.dev/) - AI capabilities
- [lingua-go](https://github.com/pemistahl/lingua-go) - Language detection

## 📞 Support

For support, please open an issue in the GitHub repository or contact the maintainers.

---

<div align="center">
Based on <a href="https://github.com/ASparkOfFire/whatsapp-livetranslate-2.0">WhatsApp LiveTranslate 2.0</a>
</div>