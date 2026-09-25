package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"atari/engine"
)

const WeightFile = "weights.bin"
const MaxMovesDefault = 200

func handleMove(w http.ResponseWriter, r *http.Request) {
	type GameRequest struct {
		Board        []int   `json:"board"`
		HumanScore   int     `json:"humanScore"`
		AiScore      int     `json:"aiScore"`
		LastMove     int     `json:"lastMove"`
		MoveCount    int     `json:"moveCount"`
		MaxMoves     int     `json:"maxMoves"`
		HumanPassed  bool    `json:"humanPassed"`
		BoardHistory [][]int `json:"boardHistory"`
	}
	type GameResponse struct {
		Move         int     `json:"move"`
		CaptureCount int     `json:"captureCount"`
		Valid        bool    `json:"valid"`
		TimeMs       float64 `json:"timeMs"`
		Nodes        int     `json:"nodes"`
		NewBoard     []int   `json:"newBoard"`
		HumanScore   int     `json:"humanScore"`
		AiScore      int     `json:"aiScore"`
		GameOver     bool    `json:"gameOver"`
		Reason       string  `json:"reason"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.MaxMoves <= 0 {
		req.MaxMoves = MaxMovesDefault
	}

	cleanBoardAfterHuman := req.Board
	humanCaptures := 0

	// 1. Фиксируем ход человека и чистим съеденные им камни
	if req.LastMove != -1 {
		boardBeforeHuman := make([]int, engine.NumCells)
		copy(boardBeforeHuman, req.Board)
		boardBeforeHuman[req.LastMove] = 0

		var legal bool
		cleanBoardAfterHuman, humanCaptures, legal = engine.CheckMoveAndCapture(boardBeforeHuman, req.LastMove, 1)
		if !legal {
			cleanBoardAfterHuman = req.Board
			humanCaptures = 0
		}
		req.HumanScore += humanCaptures
		req.MoveCount++
	} else {
		req.HumanPassed = true
		req.MoveCount++
	}

	if req.MoveCount >= req.MaxMoves {
		resp := GameResponse{GameOver: true, Reason: "Достигнут лимит ходов!", NewBoard: cleanBoardAfterHuman, HumanScore: req.HumanScore, AiScore: req.AiScore, Valid: false, Move: -1}
		json.NewEncoder(w).Encode(resp)
		return
	}

	var activeWeights []float64
	if data, err := os.ReadFile(WeightFile); err == nil && len(data) == engine.TotalParam*8 {
		dummy := NewAgent()
		if dummy.FromBytes(data) {
			activeWeights = dummy.Weights
		}
	}
	if len(activeWeights) == 0 {
		activeWeights = make([]float64, engine.TotalParam)
	}

	aiColor := -1
	tStart := time.Now()

	// 2. РАСЧЕТ ХОДА ИИ И ЖЕСТКАЯ ПРОВЕРКА КО НА ЧИСТОЙ ДОСКЕ
	bestMove, aiCaptures, finalBoard := engine.GetHybridMove(activeWeights, cleanBoardAfterHuman, aiColor, req.AiScore, req.HumanScore)

	// Если ход привел к повторению позиции (Ко), принудительно зануляем его и ищем альтернативу прямо тут
	if bestMove != -1 && len(req.BoardHistory) > 0 {
		isKo := false
		for _, past := range req.BoardHistory {
			if len(past) == engine.NumCells {
				match := true
				for i := 0; i < engine.NumCells; i++ {
					if finalBoard[i] != past[i] {
						match = false
						break
					}
				}
				if match {
					isKo = true
					break
				}
			}
		}

		// Если выскочило Ко — берем абсолютно любой другой легальный ход, кроме этого бандитского
		if isKo {
			for i := 0; i < engine.NumCells; i++ {
				if cleanBoardAfterHuman[i] == 0 && i != bestMove {
					nextB, capC, legal := engine.CheckMoveAndCapture(cleanBoardAfterHuman, i, aiColor)
					if legal {
						bestMove = i
						aiCaptures = capC
						finalBoard = nextB
						break
					}
				}
			}
		}
	}

	duration := time.Since(tStart).Seconds() * 1000.0
	req.AiScore += aiCaptures
	req.MoveCount++

	gameOver := false
	reason := ""
	if req.HumanPassed && bestMove == -1 {
		gameOver = true
		reason = "Оба игрока объявили ПАС! Игра завершена."
	} else if req.MoveCount >= req.MaxMoves {
		gameOver = true
		reason = "Достигнут лимит ходов!"
	}

	resp := GameResponse{
		Valid:        true,
		Move:         bestMove,
		CaptureCount: aiCaptures,
		TimeMs:       duration,
		Nodes:        int(engine.EvaluatedNodes),
		NewBoard:     finalBoard, // ТЕПЕРЬ СЮДА ИДЕТ СТРОГО СТЕРИЛЬНАЯ ДОСКА
		HumanScore:   req.HumanScore,
		AiScore:      req.AiScore,
		GameOver:     gameOver,
		Reason:       reason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	fs := http.FileServer(http.Dir("./webroot"))
	http.Handle("/webroot/", http.StripPrefix("/webroot/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./webroot/index.html")
	})

	http.HandleFunc("/move", handleMove)

	fmt.Printf("🚀 Рефери-сервер (Ко-Фикс) запущен на http://localhost:8080\n")
	http.ListenAndServe(":8080", nil)
}
