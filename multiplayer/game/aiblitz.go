package game

import (
	"time"

	"bryjamin.com/multiplayer/message"
	"bryjamin.com/multiplayer/send"
)

type AIBlitzGame struct {
	board     *Board
	timeLimit time.Duration
	isOver    bool
	score     int

	aiInputChannel       chan bool
	playerOutputChannel  chan *message.Message
	gameOverInputChannel chan bool
}

type AIBlitzGameState struct {
	Board          *Board
	BoardReports   []BoardReport
	Score          int
	IsOver         bool
	DestroyedPipes []DestroyedPipe
}

func NewAIBlitzGame(playerOutputChannel chan *message.Message, timeLimit time.Duration) *AIBlitzGame {

	board := NewBoard(7, 7)

	return &AIBlitzGame{
		timeLimit:            timeLimit,
		board:                &board,
		playerOutputChannel:  playerOutputChannel,
		gameOverInputChannel: make(chan bool),
		aiInputChannel:       make(chan bool),
	}

}

func (g *AIBlitzGame) Run() {

	g.board.UpdateBoardPipeConnections()

	go func() {

		send.SendMessageToAll(&SinglePlayerBlitzGameState{
			Board: g.board,
			Score: g.score,
		}, g.playerOutputChannel)

		g.aiInputChannel <- true

		for !g.isOver {
			g.timeLimit = g.timeLimit - serverTick

			send.SendMessageToAll(&TimeLimit{
				Time: g.timeLimit,
			}, g.playerOutputChannel)

			g.isOver = g.timeLimit <= 0
			if g.isOver {
				g.gameOverInputChannel <- g.isOver
			}

			time.Sleep(serverTick)
		}
	}()

OuterLoop:
	for {
		select {
		case isOver := <-g.gameOverInputChannel:
			if isOver {
				gameState := SinglePlayerBlitzGameState{
					Score:  g.score,
					IsOver: g.isOver,
				}
				send.SendMessageToAll(&gameState, g.playerOutputChannel)
				break OuterLoop
			}
		case _ = <-g.aiInputChannel:
			g.board.RotatePipeClockwise(0, 0)
			boardReports := g.board.UpdateBoardPipeConnections()

			g.score += calculateScoreFromBoardReports(boardReports)

			gameState := SinglePlayerBlitzGameState{
				BoardReports: boardReports,
				Score:        g.score,
				IsOver:       g.isOver,
			}

			send.SendMessageToAll(&gameState, g.playerOutputChannel)

			go func() { //To avoid sending too much data to the client this adds a pause before the next AI move.
				time.Sleep(serverTick / 2)
				g.aiInputChannel <- true
			}()
		}
	}

}
