package main

import (
	"math"
	"math/rand"

	"atari/engine"
)

func getPureCNNMove(a *Agent, board []int, color int) (int, int) {
	var validMoves []int
	maxCapture := 0
	var bestCaptureMoves []int

	for i := 0; i < engine.NumCells; i++ {
		_, capCount, legal := engine.CheckMoveAndCapture(board, i, color)
		if legal {
			if capCount > maxCapture {
				maxCapture = capCount
				bestCaptureMoves = []int{i}
			} else if capCount == maxCapture && capCount > 0 {
				bestCaptureMoves = append(bestCaptureMoves, i)
			}
			validMoves = append(validMoves, i)
		}
	}

	if len(validMoves) == 0 {
		return -1, 0
	}
	if maxCapture > 0 {
		return bestCaptureMoves[rand.Intn(len(bestCaptureMoves))], maxCapture
	}

	netInput := make([]float64, engine.NumCells)
	for i := 0; i < engine.NumCells; i++ {
		netInput[i] = float64(board[i] * color)
	}

	logits := engine.ForwardCNN(a.Weights, netInput)

	maxLogit := -math.MaxFloat64
	for i := 0; i < len(validMoves); i++ {
		cellLogit := logits[validMoves[i]]
		if cellLogit > maxLogit {
			maxLogit = cellLogit
		}
	}

	var bestMoves []int
	for i := 0; i < len(validMoves); i++ {
		cell := validMoves[i]
		if math.Abs(logits[cell]-maxLogit) < 0.01 {
			bestMoves = append(bestMoves, cell)
		}
	}

	if len(bestMoves) == 0 {
		return validMoves[rand.Intn(len(validMoves))], 0
	}
	return bestMoves[rand.Intn(len(bestMoves))], 0
}

func Mutate(a *Agent, rate float64, scale float64) {
	for i := range a.Weights {
		a.Weights[i] *= 0.98 // Weight Decay
		if rand.Float64() < rate {
			a.Weights[i] += rand.NormFloat64() * scale
			if a.Weights[i] > 3.0 {
				a.Weights[i] = 3.0
			}
			if a.Weights[i] < -3.0 {
				a.Weights[i] = -3.0
			}
		}
	}
}
