package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// REPL manages the interactive command loop.
type REPL struct {
	bots     []*Bot
	promptCh chan *Bot
	scanner  *bufio.Scanner
}

// NewREPL creates a new REPL instance.
func NewREPL(bots []*Bot) *REPL {
	return &REPL{
		bots:     bots,
		promptCh: make(chan *Bot, 64),
		scanner:  bufio.NewScanner(os.Stdin),
	}
}

// Run starts the REPL loop.
func (r *REPL) Run() {
	// Set up prompt channel for all bots
	for _, b := range r.bots {
		b.promptCh = r.promptCh
	}

	fmt.Println("\nTest Bot REPL ready. Type 'help' for commands.")
	fmt.Println("Bots are connected and active. Waiting for game state...")

	for {
		// Check if any bot needs manual action
		select {
		case bot := <-r.promptCh:
			if !bot.AutoPilot {
				r.handleBotPrompt(bot)
			}
		default:
		}

		fmt.Print("\n> ")
		if !r.scanner.Scan() {
			return
		}

		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}

		if r.handleCommand(line) {
			return // quit
		}
	}
}

func (r *REPL) handleCommand(line string) bool {
	parts := strings.Fields(line)
	cmd := parts[0]

	switch cmd {
	case "quit", "exit", "q":
		fmt.Println("Disconnecting bots...")
		return true

	case "help", "h":
		r.printHelp()

	case "status", "s":
		r.printStatus()

	case "start":
		if len(parts) < 2 {
			fmt.Println("Usage: start <game-name>")
			fmt.Println("Games: texas-hold-em, bourre, guts, pass-the-poop, acey-deucey, seven-card, little-l")
			return false
		}
		gameName := parts[1]
		fmt.Printf("Starting game: %s\n", gameName)
		// Use first bot (table admin) to start the game
		r.bots[0].StartGame(gameName)

	case "auto":
		allAuto := true
		for _, b := range r.bots {
			if !b.AutoPilot {
				allAuto = false
				break
			}
		}
		newState := !allAuto
		for _, b := range r.bots {
			b.AutoPilot = newState
		}
		if newState {
			fmt.Println("Auto-pilot enabled for all bots")
			r.triggerAutoPilot()
		} else {
			fmt.Println("Auto-pilot disabled for all bots")
		}

	case "act":
		if len(parts) < 2 {
			fmt.Println("Usage: act <bot-number>")
			return false
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil || idx < 1 || idx > len(r.bots) {
			fmt.Printf("Invalid bot number. Use 1-%d\n", len(r.bots))
			return false
		}
		bot := r.bots[idx-1]
		gs := bot.GetGameState()
		if gs == nil || len(gs.ValidActions) == 0 {
			fmt.Printf("Bot p%d (%s) has no pending actions\n", bot.ID, bot.Name)
			return false
		}
		r.handleBotPrompt(bot)

	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
	}

	return false
}

func (r *REPL) handleBotPrompt(bot *Bot) {
	gs := bot.GetGameState()
	if gs == nil || len(gs.ValidActions) == 0 {
		return
	}

	fmt.Printf("\n=== Bot p%d (%s) needs to act ===\n", bot.ID, bot.Name)

	// Show hand
	if len(gs.Hand) > 0 {
		cards := make([]string, len(gs.Hand))
		for i, c := range gs.Hand {
			cards[i] = c.String()
		}
		fmt.Printf("Hand: %s", strings.Join(cards, " "))
	}
	if len(gs.Community) > 0 {
		cards := make([]string, len(gs.Community))
		for i, c := range gs.Community {
			cards[i] = c.String()
		}
		fmt.Printf(" | Community: %s", strings.Join(cards, " "))
	}
	if gs.Pot > 0 {
		fmt.Printf(" | Pot: $%d", gs.Pot)
	}
	fmt.Println()
	fmt.Println()

	// Show actions
	for i, a := range gs.ValidActions {
		fmt.Printf("  [%d] %s\n", i+1, a.Name)
	}
	fmt.Println("  [a] Auto-pilot this bot")
	fmt.Println("  [A] Auto-pilot ALL bots")
	fmt.Println()

	for {
		fmt.Print("Choice: ")
		if !r.scanner.Scan() {
			return
		}

		input := strings.TrimSpace(r.scanner.Text())

		if input == "a" {
			bot.AutoPilot = true
			fmt.Printf("Bot p%d (%s) set to auto-pilot\n", bot.ID, bot.Name)
			go bot.doAutoPilot(gs)
			return
		}

		if input == "A" {
			for _, b := range r.bots {
				b.AutoPilot = true
			}
			fmt.Println("All bots set to auto-pilot")
			r.triggerAutoPilot()
			return
		}

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(gs.ValidActions) {
			fmt.Printf("Invalid choice. Enter 1-%d, 'a', or 'A'\n", len(gs.ValidActions))
			continue
		}

		chosen := gs.ValidActions[idx-1]
		ad := make(map[string]interface{})

		// Handle actions that need additional input
		switch chosen.Action {
		case actionBet, actionRaise:
			if gs.MinBet > 0 {
				fmt.Printf("Amount (%d-%d): ", gs.MinBet, gs.MaxBet)
				if !r.scanner.Scan() {
					return
				}
				amount, err := strconv.Atoi(strings.TrimSpace(r.scanner.Text()))
				if err != nil || amount < gs.MinBet || amount > gs.MaxBet {
					fmt.Printf("Invalid amount. Must be %d-%d\n", gs.MinBet, gs.MaxBet)
					continue
				}
				ad["amount"] = amount
			}
		case actionPlayCard:
			if len(chosen.Cards) > 0 {
				c := chosen.Cards[0]
				ad["cards"] = []map[string]interface{}{
					{"rank": c.Rank, "suit": c.Suit},
				}
			}
		case actionDiscard, actionTrade:
			if len(gs.Hand) > 0 {
				fmt.Println("Select cards to discard/trade (space-separated numbers, or empty for none):")
				for i, c := range gs.Hand {
					fmt.Printf("  [%d] %s\n", i+1, c.String())
				}
				fmt.Print("Cards: ")
				if !r.scanner.Scan() {
					return
				}
				cardInput := strings.TrimSpace(r.scanner.Text())
				selectedCards := make([]map[string]interface{}, 0)
				if cardInput != "" {
					for _, s := range strings.Fields(cardInput) {
						ci, err := strconv.Atoi(s)
						if err != nil || ci < 1 || ci > len(gs.Hand) {
							continue
						}
						c := gs.Hand[ci-1]
						selectedCards = append(selectedCards, map[string]interface{}{
							"rank": c.Rank, "suit": c.Suit,
						})
					}
				}
				ad["cards"] = selectedCards
			}
		case actionDecide:
			fmt.Print("Go in? (y/n): ")
			if !r.scanner.Scan() {
				return
			}
			decision := strings.ToLower(strings.TrimSpace(r.scanner.Text()))
			ad["decision"] = decision == "y" || decision == "yes"
		}

		// For pass-the-poop/acey-deucey integer actions
		if intID, err := strconv.Atoi(chosen.Action); err == nil {
			ad["id"] = intID
		}

		msg := outgoingMessage{
			Action:         chosen.Action,
			AdditionalData: ad,
		}
		bot.Send(msg)
		fmt.Printf("Sent action: %s\n", chosen.Name)
		return
	}
}

func (r *REPL) triggerAutoPilot() {
	for _, b := range r.bots {
		if b.AutoPilot {
			gs := b.GetGameState()
			if gs != nil && len(gs.ValidActions) > 0 {
				go b.doAutoPilot(gs)
			}
		}
	}
}

func (r *REPL) printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  status, s         - Show all bots and their current mode")
	fmt.Println("  start <game>      - Start a game (e.g., start bourre)")
	fmt.Println("  act <bot-number>  - Manually act for a specific bot")
	fmt.Println("  auto              - Toggle auto-pilot for all bots")
	fmt.Println("  help, h           - Show this help")
	fmt.Println("  quit, q           - Disconnect and exit")
	fmt.Println()
	fmt.Println("Game names: texas-hold-em, bourre, guts, pass-the-poop, acey-deucey, seven-card, little-l")
}

func (r *REPL) printStatus() {
	fmt.Println("\nBot Status:")
	for _, b := range r.bots {
		mode := "manual"
		if b.AutoPilot {
			mode = "auto"
		}
		actionStr := "idle"
		gs := b.GetGameState()
		if gs != nil && len(gs.ValidActions) > 0 {
			actionStr = fmt.Sprintf("%d actions pending", len(gs.ValidActions))
		}
		fmt.Printf("  p%d %-12s [%s] %s (ID: %d)\n", b.ID, b.Name, mode, actionStr, b.PlayerID)
	}
}
