package main

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"atari/engine"
)

type JudgeCandidate struct {
	Idx   int
	Score float64
}

func PlayMatch(p1, p2 *Agent) (int, int) {
	board := make([]int, engine.NumCells)
	color := 1
	p1Score, p2Score := 0, 0
	maxTurns := engine.NumCells * 2

	for turn := 0; turn < maxTurns; turn++ {
		var move, capCount int
		if color == 1 {
			move, capCount = getPureCNNMove(p1, board, 1)
			p1Score += capCount
		} else {
			move, capCount = getPureCNNMove(p2, board, -1)
			p2Score += capCount
		}

		if move == -1 {
			if color == 1 {
				return -1, turn
			}
			return 1, turn
		}

		nextBoard, _, _ := engine.CheckMoveAndCapture(board, move, color)
		board = nextBoard

		if p1Score >= engine.TargetScore {
			return 1, turn
		}
		if p2Score >= engine.TargetScore {
			return -1, turn
		}
		color = -color
	}

	if p1Score > p2Score {
		return 1, maxTurns
	}
	if p2Score > p1Score {
		return -1, maxTurns
	}
	return 0, maxTurns
}

func countTotalLibertiesMatch(board []int, color int) int {
	total := 0
	libsCounted := make([]bool, engine.NumCells)

	for i := 0; i < engine.NumCells; i++ {
		if board[i] == color {
			x, y := i/engine.BoardSize, i%engine.BoardSize
			dx := []int{-1, 1, 0, 0}
			dy := []int{0, 0, -1, 1}
			for k := 0; k < 4; k++ {
				nx, ny := x+dx[k], y+dy[k]
				if nx >= 0 && nx < engine.BoardSize && ny >= 0 && ny < engine.BoardSize {
					nIdx := nx*engine.BoardSize + ny
					if board[nIdx] == 0 && !libsCounted[nIdx] {
						total++
						libsCounted[nIdx] = true
					}
				}
			}
		}
	}
	return total
}

func evaluatePosition(board []int, judgeColor int, judgeScore, oppScore int) float64 {
	if judgeScore >= engine.TargetScore {
		return 20000.0
	}
	if oppScore >= engine.TargetScore {
		return -20000.0
	}

	judgeLibs := countTotalLibertiesMatch(board, judgeColor)
	oppLibs := countTotalLibertiesMatch(board, -judgeColor)

	return float64(judgeScore)*1000.0 - float64(oppScore)*2000.0 + float64(judgeLibs)*6.0 - float64(oppLibs)*4.0
}

func alphaBeta(board []int, depth int, alpha, beta float64, isMax bool, judgeColor, color, judgeScore, oppScore int) float64 {
	if depth == 0 {
		return evaluatePosition(board, judgeColor, judgeScore, oppScore)
	}

	validMoves := getTopCandidates(board, color, 6)
	if len(validMoves) == 0 {
		return evaluatePosition(board, judgeColor, judgeScore, oppScore)
	}

	if isMax {
		maxEval := -math.MaxFloat64
		for _, move := range validMoves {
			nextBoard, capCount, _ := engine.CheckMoveAndCapture(board, move, color)
			newJudgeScore := judgeScore
			if color == judgeColor {
				newJudgeScore += capCount
			}

			eval := alphaBeta(nextBoard, depth-1, alpha, beta, false, judgeColor, -color, newJudgeScore, oppScore)
			if eval > maxEval {
				maxEval = eval
			}
			if eval > alpha {
				alpha = eval
			}
			if beta <= alpha {
				break
			}
		}
		return maxEval
	} else {
		minEval := math.MaxFloat64
		for _, move := range validMoves {
			nextBoard, capCount, _ := engine.CheckMoveAndCapture(board, move, color)
			newOppScore := oppScore
			if color != judgeColor {
				newOppScore += capCount
			}

			eval := alphaBeta(nextBoard, depth-1, alpha, beta, true, judgeColor, -color, judgeScore, newOppScore)
			if eval < minEval {
				minEval = eval
			}
			if eval < beta {
				beta = eval
			}
			if beta <= alpha {
				break
			}
		}
		return minEval
	}
}

func getTopCandidates(board []int, color int, limit int) []int {
	var candidates []JudgeCandidate
	for i := 0; i < engine.NumCells; i++ {
		if board[i] == 0 {
			_, capCount, legal := engine.CheckMoveAndCapture(board, i, color)
			if legal {
				score := float64(capCount) * 100.0
				candidates = append(candidates, JudgeCandidate{Idx: i, Score: score})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) < limit {
		limit = len(candidates)
	}

	res := make([]int, limit)
	for i := 0; i < limit; i++ {
		res[i] = candidates[i].Idx
	}
	return res
}

func getTacticalMinimaxMove(board []int, color int, judgeScore, oppScore int, localRand *rand.Rand) int {
	bestCandidates := getTopCandidates(board, color, 12)
	if len(bestCandidates) == 0 {
		return -1
	}

	bestValue := -math.MaxFloat64
	alpha := -math.MaxFloat64
	beta := math.MaxFloat64

	type EvaluatedMove struct {
		Idx int
		Val float64
	}
	var movesWithValues []EvaluatedMove

	for _, moveIdx := range bestCandidates {
		nextBoard, capCount, _ := engine.CheckMoveAndCapture(board, moveIdx, color)
		if judgeScore+capCount >= engine.TargetScore {
			return moveIdx
		}

		boardValue := alphaBeta(nextBoard, 3, alpha, beta, false, color, -color, judgeScore+capCount, oppScore)
		movesWithValues = append(movesWithValues, EvaluatedMove{Idx: moveIdx, Val: boardValue})

		if boardValue > bestValue {
			bestValue = boardValue
		}
		if boardValue > alpha {
			alpha = boardValue
		}
	}

	var equalBestMoves []int
	for _, mv := range movesWithValues {
		if math.Abs(mv.Val-bestValue) < 0.001 {
			equalBestMoves = append(equalBestMoves, mv.Idx)
		}
	}

	// ФИКС: Возвращаем ПЕРВЫЙ элемент слайса, если тайбрейкер пуст
	if len(equalBestMoves) == 0 {
		return bestCandidates[0]
	}
	return equalBestMoves[localRand.Intn(len(equalBestMoves))]
}

func EvaluateAgainstBenchmark(alpha *Agent, totalGames int) float64 {
	benchmarkWins := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	generators := make([]*rand.Rand, totalGames)
	for g := 0; g < totalGames; g++ {
		generators[g] = rand.New(rand.NewSource(time.Now().UnixNano() + int64(g)*1234567))
	}

	for g := 0; g < totalGames; g++ {
		wg.Add(1)
		go func(gameIdx int) {
			defer wg.Done()

			localRand := generators[gameIdx]
			board := make([]int, engine.NumCells)
			color := 1
			p1Score, p2Score := 0, 0
			maxTurns := engine.NumCells * 2

			alphaColor := 1
			if gameIdx%2 != 0 {
				alphaColor = -1
			}

			for turn := 0; turn < maxTurns; turn++ {
				var move, capCount int

				if color == alphaColor {
					move, capCount = getPureCNNMove(alpha, board, alphaColor)
					if alphaColor == 1 {
						p1Score += capCount
					} else {
						p2Score += capCount
					}
				} else {
					judgeScore := p2Score
					oppScore := p1Score
					if color == 1 {
						judgeScore = p1Score
						oppScore = p2Score
					}

					move = getTacticalMinimaxMove(board, color, judgeScore, oppScore, localRand)
					if move != -1 {
						_, capCount, _ = engine.CheckMoveAndCapture(board, move, color)
						if color == 1 {
							p1Score += capCount
						} else {
							p2Score += capCount
						}
					}
				}

				if move == -1 {
					if color != alphaColor {
						mu.Lock()
						benchmarkWins++
						mu.Unlock()
					}
					break
				}

				nextBoard, _, _ := engine.CheckMoveAndCapture(board, move, color)
				board = nextBoard

				if p1Score >= engine.TargetScore {
					if alphaColor == -1 {
						mu.Lock()
						benchmarkWins++
						mu.Unlock()
					}
					break
				}
				if p2Score >= engine.TargetScore {
					if alphaColor == 1 {
						mu.Lock()
						benchmarkWins++
						mu.Unlock()
					}
					break
				}

				color = -color
			}
		}(g)
	}

	wg.Wait()
	return float64(totalGames-benchmarkWins) / float64(totalGames) * 100.0
}
