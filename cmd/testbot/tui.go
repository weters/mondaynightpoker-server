package main

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types sent from bot goroutines to the TUI via program.Send().

// BotStateMsg is sent whenever a bot receives a new game state.
type BotStateMsg struct {
	BotID int
}

// GameLogMsg is sent when a log event should be displayed.
type GameLogMsg struct {
	Message   string
	PlayerIDs []int64
}

// GameEndedMsg is sent when a game ends.
type GameEndedMsg struct {
	BotID int
}

// ErrorMsg is sent when a bot encounters an error.
type ErrorMsg struct {
	BotID   int
	Message string
}

// ClientStateMsg is sent when a clientState update is received.
type ClientStateMsg struct {
	PlayerNames map[int64]string
}

// Model is the top-level Bubble Tea model for the testbot TUI.
type Model struct {
	bots   []*Bot
	active int // index into bots[] for the currently viewed bot
	width  int
	height int

	logBuf      *LogBuffer
	playerNames map[int64]string

	// Sub-models
	overlay    OverlayModel
	gameSelect GameSelectModel
	betInput   BetInputModel
	cardSelect CardSelectModel
	inputMode  inputMode

	// Error flash
	errMsg string
}

// NewModel creates a new TUI model.
func NewModel(bots []*Bot) Model {
	return Model{
		bots:        bots,
		logBuf:      NewLogBuffer(100),
		playerNames: make(map[int64]string),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case BotStateMsg:
		m.errMsg = ""
		// Auto-focus: switch to this bot if it has actions and current bot doesn't
		botIdx := m.botIndexByID(msg.BotID)
		if botIdx >= 0 {
			bot := m.bots[botIdx]
			gs := bot.GetGameState()
			if gs != nil && len(gs.ValidActions) > 0 && !bot.AutoPilot {
				currentBot := m.bots[m.active]
				currentGS := currentBot.GetGameState()
				if currentGS == nil || len(currentGS.ValidActions) == 0 || currentBot.AutoPilot {
					m.active = botIdx
					// Clear any active input when switching
					m.inputMode = inputNone
				}
			}
		}
		return m, nil

	case ClientStateMsg:
		for id, name := range msg.PlayerNames {
			m.playerNames[id] = name
		}
		return m, nil

	case GameLogMsg:
		m.logBuf.Add(formatLogMessage(msg.Message, msg.PlayerIDs, m.playerNames))
		return m, nil

	case GameEndedMsg:
		m.logBuf.Add("Game ended")
		return m, nil

	case ErrorMsg:
		m.errMsg = msg.Message
		m.logBuf.Add(fmt.Sprintf("ERROR [p%d]: %s", msg.BotID, msg.Message))
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle ctrl+c globally
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// If game select is active, route there
	if m.gameSelect.Active {
		return m.handleGameSelectKey(msg)
	}

	// If overlay is active, route there
	if m.overlay.Active {
		return m.handleOverlayKey(msg)
	}

	// If input mode is active, route to input handler
	if m.inputMode != inputNone {
		return m.handleInputKey(msg)
	}

	// Main view key handling
	switch msg.Type {
	case tea.KeyEscape:
		m.overlay = NewOverlay(m.bots)
		return m, nil

	case tea.KeyTab:
		m.active = (m.active + 1) % len(m.bots)
		m.inputMode = inputNone
		return m, nil

	case tea.KeyShiftTab:
		m.active = (m.active - 1 + len(m.bots)) % len(m.bots)
		m.inputMode = inputNone
		return m, nil

	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= '1' && r <= '9' {
				return m.handleActionKey(int(r - '0'))
			}
		}
	}

	return m, nil
}

func (m Model) handleActionKey(num int) (tea.Model, tea.Cmd) {
	bot := m.bots[m.active]
	gs := bot.GetGameState()
	if gs == nil || len(gs.ValidActions) == 0 {
		return m, nil
	}

	idx := num - 1
	if idx >= len(gs.ValidActions) {
		return m, nil
	}

	action := gs.ValidActions[idx]

	// Check if this action needs additional input
	switch {
	case action.Action == actionBet || action.Action == actionRaise || action.NeedsAmount:
		if gs.MinBet > 0 {
			m.betInput = NewBetInput(action, gs.MinBet, gs.MaxBet)
			m.inputMode = inputBet
			return m, nil
		}
	case action.Action == actionDiscard:
		if len(gs.Hand) > 0 {
			m.cardSelect = NewCardSelect(action, gs.Hand, "Select cards to discard")
			m.inputMode = inputCardSelect
			return m, nil
		}
	case action.Action == actionTrade:
		if len(gs.Hand) > 0 {
			m.cardSelect = NewCardSelect(action, gs.Hand, "Select cards to trade")
			m.inputMode = inputCardSelect
			return m, nil
		}
	case action.Action == actionPlayCard:
		// PlayCard already has its cards embedded in the action
	case action.Action == "decide-out":
		// Send as decide with in=false
		ad := map[string]interface{}{"in": false}
		decidedAction := action
		decidedAction.Action = actionDecide
		msg := BuildMessage(gs, decidedAction, ad)
		bot.Send(msg)
		m.logBuf.Add(fmt.Sprintf("p%d %s: %s", bot.ID, bot.Name, action.Name))
		return m, nil
	case action.Action == actionDecide:
		ad := map[string]interface{}{"in": true}
		msg := BuildMessage(gs, action, ad)
		bot.Send(msg)
		m.logBuf.Add(fmt.Sprintf("p%d %s: %s", bot.ID, bot.Name, action.Name))
		return m, nil
	}

	// Simple action — send directly
	ad := make(map[string]interface{})
	if action.Action == actionPlayCard && len(action.Cards) > 0 {
		c := action.Cards[0]
		ad["cards"] = []map[string]interface{}{cardToWireFormat(c)}
	}

	outMsg := BuildMessage(gs, action, ad)
	bot.Send(outMsg)
	m.logBuf.Add(fmt.Sprintf("p%d %s: %s", bot.ID, bot.Name, action.Name))

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	bot := m.bots[m.active]
	gs := bot.GetGameState()
	if gs == nil {
		m.inputMode = inputNone
		return m, nil
	}

	switch m.inputMode {
	case inputBet:
		var amount int
		var cancel bool
		m.betInput, amount, cancel = m.betInput.Update(msg)
		if cancel {
			m.inputMode = inputNone
			return m, nil
		}
		if amount >= 0 {
			ad := map[string]interface{}{"amount": amount}
			outMsg := BuildMessage(gs, m.betInput.Action, ad)
			bot.Send(outMsg)
			m.logBuf.Add(fmt.Sprintf("p%d %s: %s $%d", bot.ID, bot.Name, m.betInput.Action.Name, amount))
			m.inputMode = inputNone
		}
		return m, nil

	case inputCardSelect:
		var selected []CardInfo
		var cancel bool
		m.cardSelect, selected, cancel = m.cardSelect.Update(msg)
		if cancel {
			m.inputMode = inputNone
			return m, nil
		}
		if selected != nil {
			ad := make(map[string]interface{})
			action := m.cardSelect.Action

			switch action.Action {
			case actionDiscard:
				wireCards := make([]map[string]interface{}, len(selected))
				for i, c := range selected {
					wireCards[i] = cardToWireFormat(c)
				}
				ad["cards"] = wireCards
			case actionTrade:
				ad["cards"] = cardInfosToDeckStrings(selected)
			}

			outMsg := BuildMessage(gs, action, ad)
			bot.Send(outMsg)
			m.logBuf.Add(fmt.Sprintf("p%d %s: %s (%d cards)", bot.ID, bot.Name, action.Name, len(selected)))
			m.inputMode = inputNone
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var action string
	m.overlay, action = m.overlay.Update(msg)

	switch {
	case action == "":
		return m, nil
	case action == "quit":
		return m, tea.Quit
	case action == "start":
		m.overlay.Active = false
		m.gameSelect = NewGameSelect()
		return m, nil
	case action == "toggle-all":
		allAuto := true
		for _, b := range m.bots {
			if !b.AutoPilot {
				allAuto = false
				break
			}
		}
		newState := !allAuto
		for _, b := range m.bots {
			b.AutoPilot = newState
		}
		if newState {
			m.logBuf.Add("Auto-pilot enabled for all bots")
			m.triggerAutoPilot()
		} else {
			m.logBuf.Add("Auto-pilot disabled for all bots")
		}
		m.overlay = NewOverlay(m.bots) // refresh
		return m, nil
	case strings.HasPrefix(action, "toggle:"):
		idStr := strings.TrimPrefix(action, "toggle:")
		for _, b := range m.bots {
			if fmt.Sprintf("%d", b.ID) == idStr {
				b.AutoPilot = !b.AutoPilot
				if b.AutoPilot {
					m.logBuf.Add(fmt.Sprintf("p%d %s: auto-pilot ON", b.ID, b.Name))
					gs := b.GetGameState()
					if gs != nil && len(gs.ValidActions) > 0 {
						go b.doAutoPilot(gs)
					}
				} else {
					m.logBuf.Add(fmt.Sprintf("p%d %s: auto-pilot OFF", b.ID, b.Name))
				}
				break
			}
		}
		m.overlay = NewOverlay(m.bots) // refresh
		return m, nil
	}

	return m, nil
}

func (m Model) handleGameSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var gameName string
	m.gameSelect, gameName = m.gameSelect.Update(msg)

	if gameName != "" {
		m.gameSelect.Active = false
		m.bots[0].StartGame(gameName)
		m.logBuf.Add(fmt.Sprintf("Starting game: %s", gameName))
	}

	return m, nil
}

func (m Model) triggerAutoPilot() {
	for _, b := range m.bots {
		if b.AutoPilot {
			gs := b.GetGameState()
			if gs != nil && len(gs.ValidActions) > 0 {
				go b.doAutoPilot(gs)
			}
		}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	innerWidth := m.width - 4 // account for border padding

	// Build sections
	header := m.renderHeader(innerWidth)
	hand := m.renderHand(innerWidth)
	community := m.renderCommunity(innerWidth)
	actions := m.renderActions(innerWidth)
	inputView := m.renderInput(innerWidth)
	logView := m.renderLogSection(innerWidth)
	statusBar := m.renderStatusBar(innerWidth)

	// Compose main view
	var sections []string
	sections = append(sections, header)
	if hand != "" {
		sections = append(sections, hand)
	}
	if community != "" {
		sections = append(sections, community)
	}
	sections = append(sections, actions)
	if inputView != "" {
		sections = append(sections, inputView)
	}
	if m.errMsg != "" {
		sections = append(sections, styleError.Render("  Error: "+m.errMsg))
	}

	mainContent := strings.Join(sections, "\n")

	// Calculate log area height
	contentLines := strings.Count(mainContent, "\n") + 1
	statusLines := 1
	borderLines := 2
	dividerLines := 1
	logLines := m.height - contentLines - statusLines - borderLines - dividerLines - 1
	if logLines < 2 {
		logLines = 2
	}

	// Build divider
	divider := styleLogDivider.Render("├" + strings.Repeat("─", innerWidth) + "┤")

	// Build log content
	logContent := logView
	// Truncate/pad log to fill space
	logActualLines := strings.Count(logContent, "\n") + 1
	if logActualLines < logLines {
		logContent += strings.Repeat("\n", logLines-logActualLines)
	}

	fullContent := mainContent + "\n" + divider + "\n" + logContent

	// Wrap in border
	bordered := styleBorder.Width(innerWidth + 2).Render(fullContent)

	// Status bar goes below the border
	result := bordered + "\n" + statusBar

	// If overlay or game select is active, render on top
	if m.gameSelect.Active {
		return m.gameSelect.View(m.width, m.height)
	}
	if m.overlay.Active {
		return m.overlay.View(m.width, m.height)
	}

	return result
}

func (m Model) renderHeader(width int) string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()

	left := styleHeader.Render(fmt.Sprintf("  p%d %s", bot.ID, bot.Name))
	if gs != nil && gs.GameName != "" {
		left += styleSectionLabel.Render(" — " + formatGameName(gs.GameName))
	}

	right := ""
	if gs != nil && gs.Pot > 0 {
		right = stylePot.Render(fmt.Sprintf("Pot: $%d  ", gs.Pot))
	}

	// Pad to fill width
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderHand(width int) string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()
	if gs == nil || len(gs.Hand) == 0 {
		return ""
	}

	label := styleSectionLabel.Render("  Your Hand:")
	cards := indentBlock(RenderHand(gs.Hand, width), "  ")
	return label + "\n" + cards
}

func (m Model) renderCommunity(width int) string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()
	if gs == nil || len(gs.Community) == 0 {
		return ""
	}

	label := styleSectionLabel.Render("  Community:")
	cards := indentBlock(RenderHand(gs.Community, width), "  ")
	return label + "\n" + cards
}

func (m Model) renderActions(width int) string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()

	if gs == nil || len(gs.ValidActions) == 0 {
		if bot.AutoPilot {
			return styleSectionLabel.Render("  Auto-pilot active — waiting for turn...")
		}
		return styleSectionLabel.Render("  Waiting for actions...")
	}

	_ = width
	parts := make([]string, len(gs.ValidActions))
	for i, a := range gs.ValidActions {
		key := styleAction.Render(fmt.Sprintf("[%d]", i+1))
		label := a.Name
		if (a.Action == actionBet || a.Action == actionRaise || a.NeedsAmount) && gs.MinBet > 0 {
			label += fmt.Sprintf(" ($%d-$%d)", gs.MinBet, gs.MaxBet)
		}
		parts[i] = key + " " + styleActionLabel.Render(label)
	}

	return "  " + strings.Join(parts, "   ")
}

func (m Model) renderInput(width int) string {
	switch m.inputMode {
	case inputBet:
		return "  " + m.betInput.View()
	case inputCardSelect:
		return "  " + m.cardSelect.View(width)
	}
	return ""
}

func (m Model) renderLogSection(width int) string {
	logLines := 5 // default
	return RenderLog(m.logBuf, logLines, width)
}

func (m Model) renderStatusBar(width int) string {
	parts := make([]string, len(m.bots))
	for i, b := range m.bots {
		gs := b.GetGameState()
		hasActions := gs != nil && len(gs.ValidActions) > 0

		label := fmt.Sprintf("p%d %s", b.ID, b.Name)
		var status string
		if i == m.active && hasActions && !b.AutoPilot {
			status = label + ":▶ACT"
			parts[i] = styleStatusActive.Render(status)
		} else if b.AutoPilot {
			status = label + ":auto"
			parts[i] = styleStatusAuto.Render(status)
		} else if hasActions {
			status = label + ":WAIT"
			parts[i] = styleStatusActive.Render(status)
		} else {
			status = label + ":idle"
			parts[i] = styleStatusIdle.Render(status)
		}
	}

	bar := " " + strings.Join(parts, " │ ")

	// Pad to width
	gap := width - lipgloss.Width(bar)
	if gap > 0 {
		bar += strings.Repeat(" ", gap)
	}

	return bar
}

func (m Model) botIndexByID(id int) int {
	for i, b := range m.bots {
		if b.ID == id {
			return i
		}
	}
	return -1
}

func formatGameName(name string) string {
	switch name {
	case gameTexasHoldEm:
		return "Texas Hold'em"
	case gameTexasHoldEmPLO:
		return "Texas Hold'em (PLO)"
	case gameSevenCard:
		return "Seven Card"
	case gameLittleL:
		return "Little L"
	case gameBourre:
		return "Bourre"
	case gameGuts:
		return "Guts"
	case gamePassThePoop:
		return "Pass the Poop"
	case gameAceyDeucey:
		return "Acey Deucey"
	default:
		return name
	}
}

// rawLogEntry represents a log message from the server.
type rawLogEntry struct {
	Message   string  `json:"message"`
	PlayerIDs []int64 `json:"playerIds"`
}

// parseLogs extracts log entries from the JSON log data.
func parseLogs(data json.RawMessage) []rawLogEntry {
	// Try array of log objects
	var logs []rawLogEntry
	if err := json.Unmarshal(data, &logs); err == nil {
		entries := make([]rawLogEntry, 0, len(logs))
		for _, l := range logs {
			if l.Message != "" {
				entries = append(entries, l)
			}
		}
		return entries
	}

	// Try single log object
	var single rawLogEntry
	if err := json.Unmarshal(data, &single); err == nil && single.Message != "" {
		return []rawLogEntry{single}
	}

	return nil
}

// parseClientState extracts player ID to display name mappings from clientState data.
func parseClientState(data json.RawMessage) map[int64]string {
	var cs map[string]struct {
		Player struct {
			DisplayName string `json:"displayName"`
		} `json:"player"`
	}
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil
	}

	names := make(map[int64]string, len(cs))
	for idStr, entry := range cs {
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil && entry.Player.DisplayName != "" {
			names[id] = entry.Player.DisplayName
		}
	}
	return names
}

// formatLogMessage replaces {} with player names and ${cents} with formatted dollar amounts.
func formatLogMessage(message string, playerIDs []int64, playerNames map[int64]string) string {
	// Replace {} with player display names
	if len(playerIDs) > 0 && playerIDs[0] != 0 {
		names := make([]string, 0, len(playerIDs))
		for _, pid := range playerIDs {
			if name, ok := playerNames[pid]; ok {
				names = append(names, name)
			} else {
				names = append(names, fmt.Sprintf("Player(%d)", pid))
			}
		}
		message = strings.ReplaceAll(message, "{}", strings.Join(names, ", "))
	}

	// Replace ${cents} with formatted dollar amounts
	message = replaceAmountTokens(message)

	return message
}

// replaceAmountTokens replaces ${cents} tokens with formatted dollar amounts.
func replaceAmountTokens(s string) string {
	var result strings.Builder
	for {
		idx := strings.Index(s, "${")
		if idx < 0 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:idx])
		rest := s[idx+2:]
		end := strings.Index(rest, "}")
		if end < 0 {
			result.WriteString(s[idx:])
			break
		}
		var cents int
		if _, err := fmt.Sscanf(rest[:end], "%d", &cents); err == nil {
			result.WriteString(formatCents(cents))
		} else {
			result.WriteString(s[idx : idx+2+end+1])
		}
		s = rest[end+1:]
	}
	return result.String()
}

// formatCents converts a cent amount to a dollar string (e.g., 150 -> "$1.50", 200 -> "$2").
func formatCents(cents int) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	dollars := cents / 100
	remainder := cents % 100
	var s string
	if remainder == 0 {
		s = fmt.Sprintf("$%d", dollars)
	} else {
		s = fmt.Sprintf("$%d.%02d", dollars, remainder)
	}
	if negative {
		s = "-" + s
	}
	return s
}

// indentBlock prepends prefix to every line of a multi-line string.
func indentBlock(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
