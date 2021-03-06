package lobby

import (
	"encoding/json"
	"log"
	"time"

	"github.com/bryjammin/pipedream/lobby/multiplayer/game"
	"github.com/bryjammin/pipedream/lobby/multiplayer/message"
	"github.com/bryjammin/pipedream/lobby/multiplayer/player"
	"github.com/bryjammin/pipedream/lobby/multiplayer/send"
)

type VersusLobby struct {
	players map[*player.Player](bool)
	game    *game.VersusPlayerBlitzGame

	register   chan *player.Player
	unregister chan *player.Player

	lobbyToPlayerMessageCh chan *player.PlayerMessage
	playerToLobbyMessageCh chan *player.PlayerMessage

	boardcastAll chan *message.Message

	isFull bool
}

func NewVersusLobby() *VersusLobby {

	return &VersusLobby{
		players:                make(map[*player.Player](bool)),
		register:               make(chan *player.Player),
		unregister:             make(chan *player.Player),
		lobbyToPlayerMessageCh: make(chan *player.PlayerMessage),
		playerToLobbyMessageCh: make(chan *player.PlayerMessage),
		boardcastAll:           make(chan *message.Message),
	}

}

func (lobby *VersusLobby) AddPlayer(p *player.Player) bool {

	if len(lobby.players) < 2 {
		lobby.players[p] = true
		if len(lobby.players) >= 2 {
			lobby.isFull = true
		}
		return true
	}
	return false
}

func (lobby *VersusLobby) RemovePlayer(p *player.Player) bool {

	if _, ok := lobby.players[p]; ok {
		delete(lobby.players, p)
		return true
	}
	return false
}

func (lobby *VersusLobby) IsFull() bool {
	return lobby.isFull
}

func (lobby *VersusLobby) UnregisterPlayer(player *player.Player) {
	lobby.unregister <- player
}

func (lobby *VersusLobby) SendMessage(message *player.PlayerMessage) {
	select {
	case lobby.playerToLobbyMessageCh <- message:
	}
}

func (lobby *VersusLobby) Run() {

	players := make([]*player.Player, len(lobby.players))
	i := 0
	for player := range lobby.players {
		player.PlayerRegister = lobby
		player.PlayerMessageReceiver = lobby
		players[i] = player
		i++
	}

	lobby.game = game.NewVersusPlayerBlitzGame(SINGLEPLAYERBLITZGAMETIMELIMIT*time.Second, players, lobby.lobbyToPlayerMessageCh, lobby.playerToLobbyMessageCh)

	go func() {
		log.Println("Beginning Versus Game...")
		send.SendMessageToAll(&LobbyBegin{
			IsStarted: true,
		}, lobby.boardcastAll)
		go lobby.game.Run()
	}()

OuterLoop:
	for {
		select {

		case unRegisteringPlayer := <-lobby.unregister:

			log.Println("Unregistering Player...")

			lobby.RemovePlayer(unRegisteringPlayer)
			if len(lobby.players) <= 0 {
				break OuterLoop
			}

		case messageToPlayer := <-lobby.lobbyToPlayerMessageCh:
			if _, ok := lobby.players[messageToPlayer.Player]; ok {
				if err := messageToPlayer.Player.WriteMessage(messageToPlayer.MessageType, messageToPlayer.Message); err != nil {
					log.Println(err)
				}
			}
		case messageFromPlayer := <-lobby.playerToLobbyMessageCh:
			var input message.BoardInput
			err := json.Unmarshal(messageFromPlayer.Message, &input)
			if err != nil {
				log.Printf("%v", err)
			}
			lobby.game.SendPlayerBoardInputToGame(&game.PlayerBoardInput{Player: messageFromPlayer.Player, BoardInput: input})
		case message := <-lobby.boardcastAll:
			for player := range lobby.players {
				if err := player.WriteMessage(message.MessageType, message.Message); err != nil {
					log.Println("Player Connection Error: ")
				}
			}
		}
	}

	log.Println("Versus Lobby Closed")

}
