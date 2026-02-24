package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	serverURL := flag.String("server", "http://localhost:5080", "server base URL")
	tableUUID := flag.String("table", "", "table UUID (creates a new table if empty)")
	numPlayers := flag.Int("players", 3, "number of bot players")
	adminEmail := flag.String("admin-email", "", "admin email for test player creation")
	adminPassword := flag.String("admin-password", "", "admin password for test player creation")
	game := flag.String("game", "", "game to auto-start after setup")
	autoPilot := flag.Bool("auto", true, "start bots in auto-pilot mode")
	flag.Parse()

	if *adminEmail == "" || *adminPassword == "" {
		fmt.Fprintln(os.Stderr, "Error: -admin-email and -admin-password are required")
		flag.Usage()
		os.Exit(1)
	}

	if *numPlayers < 2 {
		fmt.Fprintln(os.Stderr, "Error: -players must be at least 2")
		os.Exit(1)
	}

	client := NewHTTPClient(*serverURL)

	// Step 1: Login as admin
	log.Println("Logging in as admin...")
	adminJWT, err := client.Login(*adminEmail, *adminPassword)
	if err != nil {
		log.Fatalf("Admin login failed: %v", err)
	}
	log.Println("Admin login successful")

	// Step 2: Create test players
	bots := make([]*Bot, *numPlayers)
	botPassword := "testbot123"

	for i := 0; i < *numPlayers; i++ {
		name := randomName(i)
		email := fmt.Sprintf("testbot_%s_%d@example.com", name, os.Getpid())

		log.Printf("Creating test player: %s (%s)", name, email)
		playerID, err := client.CreateTestPlayer(adminJWT, name, email, botPassword)
		if err != nil {
			log.Fatalf("Failed to create test player %s: %v", name, err)
		}

		// Login the test player
		jwt, err := client.Login(email, botPassword)
		if err != nil {
			log.Fatalf("Failed to login test player %s: %v", name, err)
		}

		bots[i] = &Bot{
			ID:        i + 1,
			Name:      name,
			PlayerID:  playerID,
			JWT:       jwt,
			AutoPilot: *autoPilot,
		}

		log.Printf("  Created p%d: %s (ID: %d)", i+1, name, playerID)
	}

	// Step 3: Create or use existing table
	tblUUID := *tableUUID
	if tblUUID == "" {
		log.Println("Creating table...")
		tblUUID, err = client.CreateTable(bots[0].JWT, "Test Bot Table")
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}
		log.Printf("Table created: %s", tblUUID)
	}

	// Step 4: Join all bots to table
	for _, bot := range bots {
		log.Printf("Joining p%d (%s) to table...", bot.ID, bot.Name)
		if err := client.JoinTable(bot.JWT, tblUUID); err != nil {
			log.Fatalf("Failed to join table: %v", err)
		}
	}

	// Step 5: Connect all bots via WebSocket
	for _, bot := range bots {
		log.Printf("Connecting p%d (%s) via WebSocket...", bot.ID, bot.Name)
		if err := bot.Connect(*serverURL, tblUUID); err != nil {
			log.Fatalf("Failed to connect bot: %v", err)
		}
	}

	// Step 6: Set all bots as active
	for _, bot := range bots {
		bot.SetActive(true)
	}

	log.Printf("\nAll %d bots connected to table %s", *numPlayers, tblUUID)
	log.Printf("Table URL: %s/table/%s\n", *serverURL, tblUUID)

	// Step 7: Auto-start game if specified
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

	// Step 8: Enter REPL
	repl := NewREPL(bots)
	repl.Run()

	// Cleanup
	for _, bot := range bots {
		bot.Close()
	}
}
