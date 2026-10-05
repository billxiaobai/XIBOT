package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"
)

type config struct {
	Ad1 string `json:"ad1"`
	Ad2 string `json:"ad2"`
	Ad3 string `json:"ad3"`
	Ad4 string `json:"ad4"`
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func healthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fmt.Fprintln(w, "MC Bot is Alive!")
	})
	return mux
}

func startHealthServer() error {
	port := getenv("PORT", "3000")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen for HTTP health checks on port %s: %w", port, err)
	}

	server := &http.Server{Handler: healthHandler()}
	go func() {
		log.Printf("HTTP server running on port %s", port)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
		}
	}()
	return nil
}

func tokenPath() string {
	return getenv("BEDROCK_TOKEN_FILE", "token.txt")
}

func loadSavedToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var settings config
	if err := json.Unmarshal(data, &settings); err != nil {
		return config{}, err
	}
	return settings, nil
}

func advertisements(settings config) []string {
	ads := []string{settings.Ad1, settings.Ad2, settings.Ad3, settings.Ad4}
	result := make([]string, 0, len(ads))
	for _, ad := range ads {
		if strings.TrimSpace(ad) != "" {
			result = append(result, ad)
		}
	}
	return result
}

func renderMinecraftText(message string) string {
	const escape = "\x1b["

	colors := map[rune]string{
		'0': "30", '1': "34", '2': "32", '3': "36",
		'4': "31", '5': "35", '6': "33", '7': "37",
		'8': "90", '9': "94", 'a': "92", 'b': "96",
		'c': "91", 'd': "95", 'e': "93", 'f': "97",
	}

	var rendered strings.Builder
	runes := []rune(message)
	for index := 0; index < len(runes); index++ {
		if runes[index] != '\u00a7' || index+1 >= len(runes) {
			rendered.WriteRune(runes[index])
			continue
		}

		index++
		code := runes[index]
		switch code {
		case 'r':
			rendered.WriteString(escape + "0m")
		case 'l':
			rendered.WriteString(escape + "1m")
		case 'o':
			rendered.WriteString(escape + "3m")
		case 'n':
			rendered.WriteString(escape + "4m")
		case 'm':
			rendered.WriteString(escape + "9m")
		default:
			if color, ok := colors[code]; ok {
				rendered.WriteString(escape + color + "m")
			}
		}
	}

	return rendered.String() + escape + "0m"
}

func stripMinecraftFormatting(message string) string {
	var plain strings.Builder
	runes := []rune(message)
	for index := 0; index < len(runes); index++ {
		if runes[index] == '\u00a7' && index+1 < len(runes) {
			index++
			continue
		}
		plain.WriteRune(runes[index])
	}
	return plain.String()
}

func sendChat(conn *minecraft.Conn, message string) error {
	return conn.WritePacket(&packet.Text{
		TextType: packet.TextTypeChat,
		Message:  message,
	})
}

func startAdvertisementLoop(conn *minecraft.Conn, ads []string) {
	if len(ads) == 0 {
		log.Println("No advertisements configured")
		return
	}

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		adIndex := 0
		for range ticker.C {
			if err := sendChat(conn, ads[adIndex]); err != nil {
				log.Printf("Send advertisement failed: %v", err)
				return
			}
			fmt.Printf("[廣告] 已發送第 %d 則\n", adIndex+1)
			adIndex = (adIndex + 1) % len(ads)
		}
	}()
}

func getToken() (*oauth2.Token, error) {
	path := tokenPath()
	if token, err := loadSavedToken(path); err == nil {
		fmt.Printf("Using saved token from %s\n", path)
		return token, nil
	}

	fmt.Printf("No valid saved token found. Starting Microsoft login once and saving to %s...\n", path)
	liveToken, err := auth.RequestLiveToken()
	if err != nil {
		return nil, err
	}

	if err := saveToken(path, liveToken); err != nil {
		log.Printf("Token save failed: %v", err)
	} else {
		fmt.Printf("Saved token to %s\n", path)
	}

	return liveToken, nil
}

func main() {
	if err := startHealthServer(); err != nil {
		log.Fatalf("Start HTTP health server failed: %v", err)
	}

	serverAddr := getenv("BEDROCK_SERVER", "bedrock.mcfallout.net:19132")
	settings, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Load config failed: %v", err)
	}
	ads := advertisements(settings)

	fmt.Println("== Bedrock Bot ==")
	liveToken, err := getToken()
	if err != nil {
		log.Fatalf("Microsoft login failed: %v", err)
	}

	dialer := minecraft.Dialer{
		TokenSource: auth.RefreshTokenSource(liveToken),
	}

	fmt.Printf("Connecting to %s\n", serverAddr)
	conn, err := dialer.Dial("raknet", serverAddr)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.DoSpawn(); err != nil {
		log.Fatalf("Spawn failed: %v", err)
	}
	startAdvertisementLoop(conn, ads)

	fmt.Println("Connected and spawned. Type a message and press Enter to send it.")
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			message := scanner.Text()
			if message == "" {
				continue
			}

			if err := sendChat(conn, message); err != nil {
				log.Printf("Send message failed: %v", err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Terminal input failed: %v", err)
		}
	}()

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			log.Printf("Connection closed: %v", err)
			return
		}

		switch p := pk.(type) {
		case *packet.Text:
			plainMessage := stripMinecraftFormatting(p.Message)
			if strings.Contains(plainMessage, "xiaobai8088") {
				if err := sendChat(conn, "/tok"); err != nil {
					log.Printf("Auto-accept teleport failed: %v", err)
				} else {
					fmt.Println("[自動回覆] 已傳送 /tok")
				}
			}

			if p.SourceName != "" {
				fmt.Printf("[%s] %s\n", p.SourceName, renderMinecraftText(p.Message))
			} else {
				fmt.Println(renderMinecraftText(p.Message))
			}
		}
	}
}
