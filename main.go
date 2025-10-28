package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

const (
	announcementURL = "https://www.csie.ncu.edu.tw/announcement/category/%E6%8B%9B%E7%94%9F%E5%BF%AB%E8%A8%8A"
	checkInterval   = 1 * time.Minute
	stateFile       = "last_announcements.json"
)

var (
	telegramBotToken string
	telegramChatID   string
	discordWebhook   string
)

type Announcement struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"`
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  No .env file found, using environment variables")
	}

	// Load environment variables
	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	discordWebhook = os.Getenv("DISCORD_WEBHOOK_URL")

	fmt.Println("🚀 NCU CSIE Announcement Monitor Started")
	fmt.Printf("📍 Monitoring: %s\n", announcementURL)
	fmt.Printf("⏱️  Check interval: %v\n", checkInterval)

	// Check notification settings
	if telegramBotToken != "" && telegramChatID != "" {
		fmt.Println("✓ Telegram notifications enabled")
	} else {
		fmt.Println("✗ Telegram notifications disabled (set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID)")
	}
	if discordWebhook != "" {
		fmt.Println("✓ Discord notifications enabled")
	} else {
		fmt.Println("✗ Discord notifications disabled (set DISCORD_WEBHOOK_URL)")
	}

	fmt.Println(strings.Repeat("-", 70))

	for {
		checkForNewAnnouncements()
		time.Sleep(checkInterval)
	}
}

func sendTelegramNotification(announcement Announcement) error {
	if telegramBotToken == "" || telegramChatID == "" {
		return nil // Skip if not configured
	}

	message := fmt.Sprintf("🔔 *NCU CSIE 新公告*\n\n📢 %s\n\n🔗 %s", announcement.Title, announcement.URL)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)

	payload := map[string]interface{}{
		"chat_id":    telegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func sendDiscordNotification(announcement Announcement) error {
	if discordWebhook == "" {
		return nil // Skip if not configured
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "🔔 NCU CSIE 新公告",
				"description": announcement.Title,
				"url":         announcement.URL,
				"color":       3447003, // Blue color
				"fields": []map[string]interface{}{
					{
						"name":  "連結",
						"value": fmt.Sprintf("[點擊查看](%s)", announcement.URL),
					},
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	resp, err := http.Post(discordWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send discord message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func fetchAnnouncements() ([]Announcement, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(announcementURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	announcements := extractAnnouncements(doc)
	return announcements, nil
}

func extractAnnouncements(n *html.Node) []Announcement {
	var announcements []Announcement
	var traverse func(*html.Node)

	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "/announcement/") {
					// Get the text content of the link
					title := getTextContent(node)
					if title != "" && !strings.Contains(title, "上一頁") && !strings.Contains(title, "下一頁") {
						fullURL := attr.Val
						if !strings.HasPrefix(fullURL, "http") {
							fullURL = "https://www.csie.ncu.edu.tw" + fullURL
						}

						// Avoid duplicates
						isDuplicate := false
						for _, a := range announcements {
							if a.URL == fullURL {
								isDuplicate = true
								break
							}
						}

						if !isDuplicate && len(announcements) < 5 {
							announcements = append(announcements, Announcement{
								Title:     strings.TrimSpace(title),
								URL:       fullURL,
								Timestamp: time.Now().Format(time.RFC3339),
							})
						}
					}
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(n)
	return announcements
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getTextContent(c)
	}
	return text
}

func loadPreviousAnnouncements() ([]Announcement, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Announcement{}, nil
		}
		return nil, err
	}

	var announcements []Announcement
	if err := json.Unmarshal(data, &announcements); err != nil {
		return nil, err
	}

	return announcements, nil
}

func saveAnnouncements(announcements []Announcement) error {
	data, err := json.MarshalIndent(announcements, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

func checkForNewAnnouncements() {
	current, err := fetchAnnouncements()
	if err != nil {
		fmt.Printf("❌ Error fetching announcements: %v\n", err)
		return
	}

	previous, err := loadPreviousAnnouncements()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not load previous state: %v\n", err)
		previous = []Announcement{}
	}

	// First run
	if len(previous) == 0 {
		fmt.Printf("🔍 Initial check at %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf("   Found %d announcements. Monitoring started...\n", len(current))
		if err := saveAnnouncements(current); err != nil {
			fmt.Printf("⚠️  Warning: Could not save state: %v\n", err)
		}
		return
	}

	// Create a map of previous URLs for quick lookup
	previousURLs := make(map[string]bool)
	for _, a := range previous {
		previousURLs[a.URL] = true
	}

	// Find new announcements
	var newAnnouncements []Announcement
	for _, a := range current {
		if !previousURLs[a.URL] {
			newAnnouncements = append(newAnnouncements, a)
		}
	}

	if len(newAnnouncements) > 0 {
		fmt.Printf("\n🔔 NEW ANNOUNCEMENT(S) DETECTED at %s!\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Println(strings.Repeat("=", 70))
		for _, a := range newAnnouncements {
			fmt.Printf("\n📢 %s\n", a.Title)
			fmt.Printf("   🔗 %s\n", a.URL)

			// Send notifications
			if err := sendTelegramNotification(a); err != nil {
				fmt.Printf("⚠️  Warning: Failed to send Telegram notification: %v\n", err)
			} else if telegramBotToken != "" && telegramChatID != "" {
				fmt.Println("   ✓ Telegram notification sent")
			}

			if err := sendDiscordNotification(a); err != nil {
				fmt.Printf("⚠️  Warning: Failed to send Discord notification: %v\n", err)
			} else if discordWebhook != "" {
				fmt.Println("   ✓ Discord notification sent")
			}
		}
		fmt.Println("\n" + strings.Repeat("=", 70))

		if err := saveAnnouncements(current); err != nil {
			fmt.Printf("⚠️  Warning: Could not save state: %v\n", err)
		}
	} else {
		fmt.Printf("✓ Checked at %s - No new announcements\n", time.Now().Format("2006-01-02 15:04:05"))
	}
}
