package demoparser

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"

	log "github.com/sirupsen/logrus"
)

// Inits the players and teams.
func (p *DemoParser) handleMatchStart(events.MatchStart) {
	p.Match.Map = p.Match.Header.MapName
	p.SidesSwitched = false

	if ConfigData.IsDebug() {
		p.debug(fmt.Sprintf("Game started on map %v", p.Match.Map))
	}

	gameState := p.parser.GameState()

	// Create teams.
	ct := gameState.TeamCounterTerrorists()
	t := gameState.TeamTerrorists()

	teams := p.Match.Teams

	teams[GetTeamIndex(t.Team(), p.SidesSwitched)] = &Team{State: t, StartedAs: common.TeamTerrorists}
	teams[GetTeamIndex(ct.Team(), p.SidesSwitched)] = &Team{State: ct, StartedAs: common.TeamCounterTerrorists}

	// Create players and map them to the teams.
	for _, player := range gameState.Participants().Playing() {
		if player.IsBot {
			continue
		}

		teamID := GetTeamIndex(player.Team, p.SidesSwitched)
		teamPlayers := p.Match.Teams[teamID].Players

		customPlayer := &Player{SteamID: player.SteamID64, Name: player.Name, Team: teams[teamID]}

		teamPlayers = append(teamPlayers, customPlayer)
	}
}

func (p *DemoParser) handleGamePhaseChanged(e events.GamePhaseChanged) {
	switch e.NewGamePhase {
	case common.GamePhaseInit:
		p.SidesSwitched = false
	case common.GamePhaseTeamSideSwitch:
		p.SidesSwitched = !p.SidesSwitched
	case common.GamePhaseGameEnded:
		p.Match.Duration = p.parser.CurrentTime()
	}
}

func (p *DemoParser) handleRoundStart(e events.RoundStart) {
	p.CurrentRound++
	p.RoundOngoing = true
	p.RoundStart = p.parser.CurrentTime()
	p.Match.Rounds = append(p.Match.Rounds, &Round{})

	if ConfigData.IsDebug() {
		p.debug(fmt.Sprintf("Starting round %d", p.CurrentRound))
	}
}

func (p *DemoParser) handleMVP(e events.RoundMVPAnnouncement) {
	player, err := p.getPlayer(e.Player)
	if err != nil {
		log.Panic(err)
	}

	if ConfigData.IsDebug() {
		p.debug(fmt.Sprintf("MVP for round %d is %v", p.CurrentRound, player.Name))
	}

	p.Match.Rounds[p.CurrentRound-1].MVP = player
}

func (p *DemoParser) handleRoundEnd(e events.RoundEnd) {
	if !p.RoundOngoing {
		return
	}

	p.RoundOngoing = false
	round := p.Match.Rounds[p.CurrentRound-1]

	if ConfigData.IsDebug() {
		p.debug(fmt.Sprintf("Ending round %d with winner %v", p.CurrentRound, e.Message))
	}

	round.Winner = p.Match.Teams[GetTeamIndex(e.Winner, p.SidesSwitched)]
	round.WinReason = e.Reason
	round.Duration = p.parser.CurrentTime() - p.RoundStart
}

func (p *DemoParser) handleKill(e events.Kill) {
	// Ignore warm-up kills
	if p.CurrentRound == 0 {
		return
	}

	victim, err := p.getPlayer(e.Victim)
	if err != nil {
		// TODO: This happens -> Either the victim disconnected or there is no victim? The steam id is not found in the structs.
		// Create issue
		log.Panic(err)
	}

	round := p.Match.Rounds[p.CurrentRound-1]
	kill := &Kill{Time: p.parser.CurrentTime(), Weapon: e.Weapon.Type, IsHeadshot: e.IsHeadshot, Victim: victim}
	round.Kills = append(round.Kills, kill)

	// Add optional killer if player died e.g. through fall damage
	if e.Killer != nil {
		killer, err := p.getPlayer(e.Killer)
		if err != nil {
			log.Panic(err)
		}
		kill.Killer = killer
	}

	// Add optional assister
	if e.Assister != nil {
		assister, err := p.getPlayer(e.Assister)
		if err != nil {
			log.Panic(err)
		}
		kill.Assister = assister
	}
}

func (p *DemoParser) debug(message string) {
	if ConfigData.IsTrace() {
		log.WithFields(log.Fields{
			"Match": p.Match.ID,
			"Round": p.CurrentRound,
		}).Trace(message)
	} else {
		log.Debug(message)
	}
}
