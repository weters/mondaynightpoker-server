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

// BotConnMsg is sent when a bot's connection drops or is restored.
type BotConnMsg struct {
	BotID     int
	Connected bool
}

// Model is the top-level Bubble Tea model for the testbot TUI.
type Model struct {
	bots   []*Bot
	active int // index into bots[] for the currently viewed bot
	width  int
	height int

	logBuf      *LogBuffer
	playerNames map[int64]string

	// lastActorBotID is the ID of the bot currently "up to act" — the one the
	// view has auto-snapped to. While the same actor is still up, manual tabs
	// stick; when the actor changes (i.e., the previous one has lost their
	// valid actions), the view jumps to the new actor. Zero means no actor.
	lastActorBotID int

	// Sub-models
	overlay    OverlayModel
	gameSelect GameSelectModel
	betInput   BetInputModel
	cardSelect CardSelectModel
	inputMode  inputMode

	// View toggles
	showDashboard bool
	showHelp      bool

	// logScroll is how many entries the log view is scrolled up from the
	// live tail; 0 means following new entries.
	logScroll int

	// lastGame is the most recently seen game name, used by the restart key.
	lastGame string

	// Error flash
	errMsg string
}

// logScrollStep is how many entries PgUp/PgDn move the log view.
const logScrollStep = 5

// NewModel creates a new TUI model.
func NewModel(bots []*Bot) Model {
	return Model{
		bots:        bots,
		logBuf:      NewLogBuffer(500),
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
		botIdx := m.botIndexByID(msg.BotID)
		if botIdx < 0 {
			return m, nil
		}

		// Re-check whether the previously-tracked actor is still acting.
		// If they no longer have actions (or went on auto-pilot), they've
		// yielded the turn, so clear them and let the next actor claim it.
		if m.lastActorBotID != 0 && !m.botIsActor(m.lastActorBotID) {
			m.lastActorBotID = 0
		}

		bot := m.bots[botIdx]
		gs := bot.GetGameState()
		if gs != nil && gs.GameName != "" {
			m.lastGame = gs.GameName
		}
		hasActions := gs != nil && len(gs.ValidActions) > 0 && !bot.AutoPilot

		// If no one is currently the actor and this bot has actions, claim
		// them and snap focus. While the same actor is up, repeated state
		// updates leave m.active alone — the user keeps any manual tab.
		if hasActions && m.lastActorBotID == 0 {
			m.active = botIdx
			m.lastActorBotID = msg.BotID
			m.inputMode = inputNone
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

	case BotConnMsg:
		botIdx := m.botIndexByID(msg.BotID)
		if botIdx < 0 {
			return m, nil
		}
		bot := m.bots[botIdx]
		if msg.Connected {
			m.logBuf.Add(fmt.Sprintf("p%d %s: reconnected", bot.ID, bot.Name))
		} else {
			m.logBuf.Add(fmt.Sprintf("p%d %s: connection lost, reconnecting...", bot.ID, bot.Name))
		}
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

	// Help is modal: any key dismisses it
	if m.showHelp {
		m.showHelp = false
		return m, nil
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

	case tea.KeyPgUp:
		m.logScroll += logScrollStep
		if limit := m.logBuf.Len(); m.logScroll > limit {
			m.logScroll = limit
		}
		return m, nil

	case tea.KeyPgDown:
		m.logScroll -= logScrollStep
		if m.logScroll < 0 {
			m.logScroll = 0
		}
		return m, nil

	case tea.KeyEnd:
		m.logScroll = 0
		return m, nil

	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			return m.handleRuneKey(msg.Runes[0])
		}
	}

	return m, nil
}

// handleRuneKey processes single-character shortcuts in the main view.
func (m Model) handleRuneKey(r rune) (tea.Model, tea.Cmd) {
	switch {
	case r >= '1' && r <= '9':
		return m.handleActionKey(int(r - '0'))

	case r == '?':
		m.showHelp = true

	case r == 'd':
		m.showDashboard = !m.showDashboard

	case r == 'a':
		m.toggleAutoPilot(m.bots[m.active])

	case r == 'A':
		allAuto := true
		for _, b := range m.bots {
			if !b.AutoPilot {
				allAuto = false
				break
			}
		}
		m.setAllAutoPilot(!allAuto)

	case r == 's':
		speed := cycleSpeed()
		m.logBuf.Add(fmt.Sprintf("Auto-pilot speed: %s", speed))

	case r == 'g':
		m.gameSelect = NewGameSelect()

	case r == 'r':
		if m.lastGame == "" {
			m.errMsg = "no game to restart yet"
			return m, nil
		}
		m.bots[0].StartGame(m.lastGame)
		m.logBuf.Add(fmt.Sprintf("Restarting game: %s", formatGameName(m.lastGame)))

	case r == 'T':
		m.bots[0].TerminateGame()
		m.logBuf.Add("Terminating game...")
	}

	return m, nil
}

// toggleAutoPilot flips auto-pilot for a single bot and kicks it if it has
// pending actions.
func (m Model) toggleAutoPilot(b *Bot) {
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
}

// setAllAutoPilot sets auto-pilot for every bot and kicks any with pending
// actions when enabling.
func (m Model) setAllAutoPilot(on bool) {
	for _, b := range m.bots {
		b.AutoPilot = on
	}
	if on {
		m.logBuf.Add("Auto-pilot enabled for all bots")
		m.triggerAutoPilot()
	} else {
		m.logBuf.Add("Auto-pilot disabled for all bots")
	}
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
	case action.Action == actionFold && gs.GameName == gameBourre:
		// Bourre fold sends discard with nil cards
		msg := outgoingMessage{Action: actionDiscard}
		bot.Send(msg)
		m.logBuf.Add(fmt.Sprintf("p%d %s: Fold", bot.ID, bot.Name))
		return m, nil
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
	case action == "terminate":
		m.overlay.Active = false
		m.bots[0].TerminateGame()
		m.logBuf.Add("Terminating game...")
		return m, nil
	case action == "cancel-pending":
		m.overlay.Active = false
		m.bots[0].CancelGame()
		m.logBuf.Add("Cancelling pending game...")
		return m, nil
	case action == "toggle-all":
		allAuto := true
		for _, b := range m.bots {
			if !b.AutoPilot {
				allAuto = false
				break
			}
		}
		m.setAllAutoPilot(!allAuto)
		m.overlay = NewOverlay(m.bots) // refresh
		return m, nil
	case strings.HasPrefix(action, "toggle:"):
		idStr := strings.TrimPrefix(action, "toggle:")
		for _, b := range m.bots {
			if fmt.Sprintf("%d", b.ID) == idStr {
				m.toggleAutoPilot(b)
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
		m.lastGame = gameName
		m.logBuf.Add(fmt.Sprintf("Starting game: %s", formatGameName(gameName)))
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

	// Modals take over the whole screen
	if m.showHelp {
		return RenderHelp(m.width, m.height)
	}
	if m.gameSelect.Active {
		return m.gameSelect.View(m.width, m.height)
	}
	if m.overlay.Active {
		return m.overlay.View(m.width, m.height)
	}

	header := m.renderHeaderBar(m.width)
	tabs := m.renderTabs()
	footer := m.renderStatusBar(m.width)

	actionsContent := m.renderActionsContent()
	actionsHeight := lipgloss.Height(actionsContent) + 2

	// header + tabs + footer = 3 lines
	mainHeight := m.height - 3 - actionsHeight
	if mainHeight < 5 {
		mainHeight = 5
	}

	tableTitle := "Table"
	if m.showDashboard {
		tableTitle = "Dashboard"
	}

	var mainArea string
	if m.width >= 90 {
		// Wide: table and log side by side
		logWidth := m.width * 2 / 5
		if logWidth > 60 {
			logWidth = 60
		}
		tableWidth := m.width - logWidth
		tablePanel := panel(tableTitle, m.renderTableContent(tableWidth-4), tableWidth, mainHeight, true)
		logPanel := panel("Log", m.renderLogContent(mainHeight-2, logWidth-4), logWidth, mainHeight, false)
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, tablePanel, logPanel)
	} else {
		// Narrow: table stacked above log
		logHeight := mainHeight * 2 / 5
		if logHeight < 4 {
			logHeight = 4
		}
		tableHeight := mainHeight - logHeight
		tablePanel := panel(tableTitle, m.renderTableContent(m.width-4), m.width, tableHeight, true)
		logPanel := panel("Log", m.renderLogContent(logHeight-2, m.width-4), m.width, logHeight, false)
		mainArea = tablePanel + "\n" + logPanel
	}

	bot := m.bots[m.active]
	actionsTitle := fmt.Sprintf("Actions — p%d %s", bot.ID, bot.Name)
	actionsPanel := panel(actionsTitle, actionsContent, m.width, actionsHeight, m.inputMode != inputNone)

	return strings.Join([]string{header, tabs, mainArea, actionsPanel, footer}, "\n")
}

// renderHeaderBar renders the full-width title bar with the current game and pot.
func (m Model) renderHeaderBar(width int) string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()

	left := " ♠ MONDAY NIGHT POKER · TESTBOT"
	if gs != nil && gs.GameName != "" {
		left += "  —  " + formatGameName(gs.GameName)
	}

	right := ""
	if gs != nil && gs.Pot > 0 {
		right = "POT " + formatCents(gs.Pot) + " "
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return styleAppHeader.Render(left + strings.Repeat(" ", gap) + right)
}

// renderTabs renders one tab per bot; the focused bot is highlighted and
// bots with pending actions get an alert tab.
func (m Model) renderTabs() string {
	parts := make([]string, len(m.bots))
	for i, b := range m.bots {
		label := fmt.Sprintf("p%d %s", b.ID, b.Name)
		gs := b.GetGameState()
		needsAct := gs != nil && len(gs.ValidActions) > 0 && !b.AutoPilot

		switch {
		case i == m.active:
			parts[i] = styleTabActive.Render(label)
		case needsAct:
			parts[i] = styleTabAlert.Render(label)
		default:
			parts[i] = styleTabInactive.Render(label)
		}
	}

	return " " + strings.Join(parts, " ")
}

// renderTableContent renders the focused bot's cards (or the dashboard).
func (m Model) renderTableContent(width int) string {
	if m.showDashboard {
		return RenderDashboard(m.bots, m.active)
	}

	bot := m.bots[m.active]
	gs := bot.GetGameState()
	if gs == nil {
		return styleSectionLabel.Render("No game in progress") + "\n\n" +
			styleFooterHint.Render("Press ") + styleFooterKey.Render("g") + styleFooterHint.Render(" to start a game, or ") +
			styleFooterKey.Render("?") + styleFooterHint.Render(" for help.")
	}

	var sections []string
	if len(gs.Hand) > 0 {
		sections = append(sections, styleSectionLabel.Render("Your Hand")+"\n"+RenderHand(gs.Hand, width))
	}
	if len(gs.Community) > 0 {
		sections = append(sections, styleSectionLabel.Render("Community")+"\n"+RenderHand(gs.Community, width))
	}
	if gs.TrumpCard != nil {
		sections = append(sections, styleSectionLabel.Render("Trump  ")+RenderHandInline([]CardInfo{*gs.TrumpCard}))
	}
	if len(gs.AceyCards) > 0 {
		sections = append(sections, styleSectionLabel.Render("Board  ")+RenderHandInline(gs.AceyCards))
	}

	if len(sections) == 0 {
		return styleSectionLabel.Render("Waiting for cards...")
	}

	return strings.Join(sections, "\n\n")
}

// renderLogContent renders the scrollable log for the log panel.
func (m Model) renderLogContent(lines, width int) string {
	return RenderLogWindow(m.logBuf, lines, width, m.logScroll)
}

// renderActionsContent renders the action chips, any active input, and error flash.
func (m Model) renderActionsContent() string {
	bot := m.bots[m.active]
	gs := bot.GetGameState()

	var lines []string
	if gs == nil || len(gs.ValidActions) == 0 {
		if bot.AutoPilot {
			lines = append(lines, styleStatusAuto.Render("Auto-pilot active — waiting for turn..."))
		} else {
			lines = append(lines, styleSectionLabel.Render("Waiting for actions..."))
		}
	} else {
		parts := make([]string, len(gs.ValidActions))
		for i, a := range gs.ValidActions {
			label := a.Name
			if (a.Action == actionBet || a.Action == actionRaise || a.NeedsAmount) && gs.MinBet > 0 {
				label += fmt.Sprintf(" ($%d-$%d)", gs.MinBet, gs.MaxBet)
			}
			parts[i] = styleAction.Render(fmt.Sprintf("%d", i+1)) + " " + styleActionLabel.Render(label)
		}
		lines = append(lines, strings.Join(parts, "  "))
	}

	switch m.inputMode {
	case inputBet:
		lines = append(lines, m.betInput.View())
	case inputCardSelect:
		lines = append(lines, m.cardSelect.View(0))
	}

	if m.errMsg != "" {
		lines = append(lines, styleError.Render("✖ "+m.errMsg))
	}

	return strings.Join(lines, "\n")
}

// renderStatusBar renders the footer: per-bot status dots and key hints.
func (m Model) renderStatusBar(width int) string {
	parts := make([]string, len(m.bots))
	for i, b := range m.bots {
		gs := b.GetGameState()
		hasActions := gs != nil && len(gs.ValidActions) > 0

		label := fmt.Sprintf("p%d %s", b.ID, b.Name)
		switch {
		case b.Disconnected():
			parts[i] = styleError.Render("● " + label + " off")
		case b.AutoPilot:
			parts[i] = styleStatusAuto.Render("● " + label + " auto")
		case hasActions && i == m.active:
			parts[i] = styleStatusActive.Render("● " + label + " ACT")
		case hasActions:
			parts[i] = styleStatusActive.Render("● " + label + " WAIT")
		default:
			parts[i] = styleStatusIdle.Render("○ " + label)
		}
	}

	bar := " " + strings.Join(parts, "  ")
	right := styleFooterKey.Render("d") + styleFooterHint.Render(" dash · ") +
		styleFooterKey.Render("g") + styleFooterHint.Render(" game · ") +
		styleFooterHint.Render(fmt.Sprintf("speed:%s · ", currentSpeed())) +
		styleFooterKey.Render("?:help") + " "

	// Pad between left and right
	gap := width - lipgloss.Width(bar) - lipgloss.Width(right)
	if gap > 0 {
		bar += strings.Repeat(" ", gap) + right
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

// botIsActor reports whether the bot with the given ID currently has valid
// actions and is not on auto-pilot — i.e., a human at the TUI is expected to
// act for them.
func (m Model) botIsActor(id int) bool {
	idx := m.botIndexByID(id)
	if idx < 0 {
		return false
	}
	bot := m.bots[idx]
	if bot.AutoPilot {
		return false
	}
	gs := bot.GetGameState()
	return gs != nil && len(gs.ValidActions) > 0
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
