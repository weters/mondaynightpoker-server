package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func main() {
	serverURL := flag.String("server", "http://localhost:5080", "server base URL")
	tableUUID := flag.String("table", "", "table UUID (creates a new table if empty)")
	numPlayers := flag.Int("players", 3, "number of bot players to create (ignored with -join)")
	adminEmail := flag.String("admin-email", "", "admin email for test player creation")
	adminPassword := flag.String("admin-password", "", "admin password for test player creation")
	game := flag.String("game", "", "game to auto-start after setup")
	autoPilot := flag.Bool("auto", false, "start bots in auto-pilot mode")
	join := flag.Bool("join", false, "join existing table players instead of creating new ones (requires -table)")
	speed := flag.String("speed", "normal", "auto-pilot speed: instant, fast, normal, or slow")
	flag.Parse()

	if !setSpeed(*speed) {
		fmt.Fprintf(os.Stderr, "Error: unknown -speed %q (want instant, fast, normal, or slow)\n", *speed)
		os.Exit(1)
	}

	if *adminEmail == "" || *adminPassword == "" {
		fmt.Fprintln(os.Stderr, "Error: -admin-email and -admin-password are required")
		flag.Usage()
		os.Exit(1)
	}

	if *join && *tableUUID == "" {
		fmt.Fprintln(os.Stderr, "Error: -join requires -table")
		os.Exit(1)
	}

	if !*join && *numPlayers < 2 {
		fmt.Fprintln(os.Stderr, "Error: -players must be at least 2")
		os.Exit(1)
	}

	client := NewHTTPClient(*serverURL)

	// Login as admin
	log.Println("Logging in as admin...")
	adminJWT, adminPlayerID, err := client.Login(*adminEmail, *adminPassword)
	if err != nil {
		log.Fatalf("Admin login failed: %v", err)
	}
	log.Println("Admin login successful")

	var bots []*Bot
	var tblUUID string

	if *join {
		tblUUID = *tableUUID
		bots, err = joinExistingTable(client, adminJWT, adminPlayerID, tblUUID, *autoPilot)
		if err != nil {
			log.Fatalf("Failed to join existing table: %v", err)
		}
	} else {
		bots, tblUUID, err = createNewBots(client, adminJWT, *tableUUID, *numPlayers, *autoPilot)
		if err != nil {
			log.Fatalf("Failed to create bots: %v", err)
		}
	}

	if len(bots) == 0 {
		log.Fatal("No players to control")
	}

	// Connect all bots via WebSocket (only first bot forwards logs to avoid duplicates)
	for i, bot := range bots {
		bot.forwardLogs = i == 0
		log.Printf("Connecting p%d (%s) via WebSocket...", bot.ID, bot.Name)
		if err := bot.Connect(*serverURL, tblUUID); err != nil {
			log.Fatalf("Failed to connect bot: %v", err)
		}
	}

	// Set all bots as active
	for _, bot := range bots {
		bot.SetActive(true)
	}

	log.Printf("\nAll %d bots connected to table %s", len(bots), tblUUID)
	log.Printf("Table URL: %s/table/%s\n", *serverURL, tblUUID)

	// Auto-start game if specified
	if *game != "" {
		log.Printf("Starting game: %s", *game)
		bots[0].StartGame(*game)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		for _, bot := range bots {
			bot.Close()
		}
		os.Exit(0)
	}()

	// Create and run the TUI. zone.NewGlobal initializes the bubblezone
	// manager used for mouse hit-testing; WithMouseCellMotion enables mouse
	// reporting so clicks and wheel events reach Update.
	zone.NewGlobal()
	m := NewModel(bots)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Wire up tea.Program to all bots so they can send messages
	for _, bot := range bots {
		bot.program = p
	}

	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	// Cleanup
	for _, bot := range bots {
		bot.Close()
	}
}

// joinExistingTable fetches players from an existing table, resets their passwords
// via admin, logs in as each, and returns bots ready to connect.
// It skips the admin player (adminPlayerID) to avoid resetting the admin's password.
func joinExistingTable(client *HTTPClient, adminJWT string, adminPlayerID int64, tableUUID string, autoPilot bool) ([]*Bot, error) {
	log.Printf("Fetching players from table %s...", tableUUID)
	players, err := client.GetTablePlayers(adminJWT, tableUUID)
	if err != nil {
		return nil, err
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no players found at table %s", tableUUID)
	}

	botPassword := "testbot_takeover"
	bots := make([]*Bot, 0, len(players))

	for i, p := range players {
		playerID := p.Player.ID

		if playerID == adminPlayerID {
			log.Printf("Skipping admin player: %s (ID: %d)", p.Player.DisplayName, playerID)
			continue
		}
		name := p.Player.DisplayName

		log.Printf("Taking control of player: %s (ID: %d)", name, playerID)

		// Reset password via admin
		if err := client.SetPlayerPassword(adminJWT, playerID, botPassword); err != nil {
			return nil, fmt.Errorf("reset password for %s: %w", name, err)
		}

		// Look up email via admin search, then login
		jwt, err := loginByPlayerID(client, adminJWT, playerID, botPassword)
		if err != nil {
			return nil, fmt.Errorf("login as %s: %w", name, err)
		}

		bots = append(bots, &Bot{
			ID:        i + 1,
			Name:      name,
			PlayerID:  playerID,
			JWT:       jwt,
			AutoPilot: autoPilot,
		})

		log.Printf("  Controlling p%d: %s (ID: %d)", i+1, name, playerID)
	}

	return bots, nil
}

type adminPlayerWithEmail struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

// loginByPlayerID uses the admin player search to find the player's email, then logs in.
func loginByPlayerID(client *HTTPClient, adminJWT string, playerID int64, password string) (string, error) {
	var players []adminPlayerWithEmail
	path := fmt.Sprintf("/player?search=%d&start=0&rows=1", playerID)
	if err := client.getJSON(path, adminJWT, &players); err != nil {
		return "", fmt.Errorf("admin player search: %w", err)
	}

	if len(players) == 0 {
		return "", fmt.Errorf("player %d not found via admin search", playerID)
	}

	jwt, _, err := client.Login(players[0].Email, password)
	return jwt, err
}

// createNewBots creates test players, a table if needed, and joins them.
func createNewBots(client *HTTPClient, adminJWT, tableUUID string, numPlayers int, autoPilot bool) ([]*Bot, string, error) {
	bots := make([]*Bot, numPlayers)
	botPassword := "testbot123"

	for i := 0; i < numPlayers; i++ {
		name := randomName(i)
		email := fmt.Sprintf("testbot_%s_%d@example.com", name, os.Getpid())

		log.Printf("Creating test player: %s (%s)", name, email)
		playerID, err := client.CreateTestPlayer(adminJWT, name, email, botPassword)
		if err != nil {
			return nil, "", fmt.Errorf("create test player %s: %w", name, err)
		}

		jwt, _, err := client.Login(email, botPassword)
		if err != nil {
			return nil, "", fmt.Errorf("login test player %s: %w", name, err)
		}

		bots[i] = &Bot{
			ID:        i + 1,
			Name:      name,
			PlayerID:  playerID,
			JWT:       jwt,
			AutoPilot: autoPilot,
		}

		log.Printf("  Created p%d: %s (ID: %d)", i+1, name, playerID)
	}

	tblUUID := tableUUID
	if tblUUID == "" {
		log.Println("Creating table...")
		var err error
		tblUUID, err = client.CreateTable(bots[0].JWT, "Test Bot Table")
		if err != nil {
			return nil, "", fmt.Errorf("create table: %w", err)
		}
		log.Printf("Table created: %s", tblUUID)
	}

	for _, bot := range bots {
		log.Printf("Joining p%d (%s) to table...", bot.ID, bot.Name)
		if err := client.JoinTable(bot.JWT, tblUUID); err != nil {
			return nil, "", fmt.Errorf("join table: %w", err)
		}
	}

	return bots, tblUUID, nil
}
