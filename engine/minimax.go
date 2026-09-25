package engine

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

type CacheEntry struct {
	Hash  uint64
	Depth int
	Value float64
	Flag  int // 0 = Exact, 1 = Alpha (Upper), 2 = Beta (Lower)
}

// ОБЪЯВЛЕНИЕ ПРОПУЩЕННОЙ СТРУКТУРЫ
type CandidateMove struct {
	Idx   int
	Score float64
}

const CacheSize = 1048576
const CacheMask = CacheSize - 1

var GlobalCache []CacheEntry
var ZobristBlack [NumCells]uint64
var ZobristWhite [NumCells]uint64
var zobristInitDone bool

var EvaluatedNodes int

func initZobrist() {
	if zobristInitDone {
		return
	}
	r := rand.New(rand.NewSource(1337))
	for i := 0; i < NumCells; i++ {
		ZobristBlack[i] = r.Uint64()
		ZobristWhite[i] = r.Uint64()
	}
	GlobalCache = make([]CacheEntry, CacheSize)
	zobristInitDone = true
}

func computeZobristHash(board []int, aiScore, humanScore int) uint64 {
	initZobrist()
	var h uint64
	for i := 0; i < NumCells; i++ {
		if board[i] == 1 {
			h ^= ZobristBlack[i]
		} else if board[i] == -1 {
			h ^= ZobristWhite[i]
		}
	}
	h ^= uint64(aiScore) * 99991
	h ^= uint64(humanScore) * 12433
	return h
}

func evaluatePurePosition(board []int, aiColor, aiScore, humanScore int) float64 {
	EvaluatedNodes++
	if aiScore >= TargetScore {
		return 1000000.0
	}
	if humanScore >= TargetScore {
		return -1000000.0
	}

	myTotalLibs := 0
	oppTotalLibs := 0
	myAtariCount := 0
	oppAtariCount := 0

	visited := make([]bool, NumCells)

	for i := 0; i < NumCells; i++ {
		if board[i] != 0 && !visited[i] {
			color := board[i]
			group, liberties := getGroupStats(board, i, color)
			for _, member := range group {
				visited[member] = true
			}

			if color == aiColor {
				myTotalLibs += liberties
				if liberties == 1 {
					myAtariCount += len(group)
				}
			} else {
				oppTotalLibs += liberties
				if liberties == 1 {
					oppAtariCount += len(group)
				}
			}
		}
	}

	return float64(aiScore)*100000.0 - float64(humanScore)*200000.0 - float64(myAtariCount)*50000.0 + float64(oppAtariCount)*25000.0 + float64(myTotalLibs)*2000.0 - float64(oppTotalLibs)*1000.0
}

func getOrderedMoves(board []int, color int, maxWidth int) []int {
	var list []CandidateMove
	for i := 0; i < NumCells; i++ {
		if board[i] == 0 {
			nextBoard, capCount, legal := CheckMoveAndCapture(board, i, color)
			if legal {
				score := float64(capCount) * 10000.0
				_, moveLiberties := getGroupStats(nextBoard, i, color)
				score += float64(moveLiberties) * 100.0

				if isCellInAtari(nextBoard, i, -color) {
					score += 500.0
				}
				if capCount == 0 && moveLiberties == 1 {
					score -= 50000.0
				}

				list = append(list, CandidateMove{Idx: i, Score: score})
			}
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
	if len(list) < maxWidth {
		maxWidth = len(list)
	}
	res := make([]int, maxWidth)
	for i := 0; i < maxWidth; i++ {
		res[i] = list[i].Idx
	}
	return res
}

func hybridAlphaBeta(currentHash uint64, board []int, depth int, alpha, beta float64, isMax bool, aiColor, color, aiScore, humanScore int) float64 {
	cacheIdx := currentHash & CacheMask
	entry := GlobalCache[cacheIdx]

	if entry.Hash == currentHash && entry.Depth >= depth {
		if entry.Flag == 0 {
			return entry.Value
		}
		if entry.Flag == 1 && entry.Value > alpha {
			alpha = entry.Value
		}
		if entry.Flag == 2 && entry.Value < beta {
			beta = entry.Value
		}
		if alpha >= beta {
			return entry.Value
		}
	}

	if depth == 0 {
		val := evaluatePurePosition(board, aiColor, aiScore, humanScore)
		GlobalCache[cacheIdx] = CacheEntry{Hash: currentHash, Depth: depth, Value: val, Flag: 0}
		return val
	}

	validMoves := getOrderedMoves(board, color, 6)
	if len(validMoves) == 0 {
		validMoves = []int{-1}
	}
	originalAlpha := alpha

	if isMax {
		maxEval := -math.MaxFloat64
		for _, move := range validMoves {
			var nextBoard []int
			var capCount int
			var nextHash uint64
			if move == -1 {
				nextBoard = make([]int, NumCells)
				copy(nextBoard, board)
				nextHash = currentHash
			} else {
				nextBoard, capCount, _ = CheckMoveAndCapture(board, move, color)
				nextHash = currentHash ^ ZobristBlack[move]
			}
			newAiScore := aiScore
			if color == aiColor {
				newAiScore += capCount
			}

			eval := hybridAlphaBeta(nextHash, nextBoard, depth-1, alpha, beta, false, aiColor, -color, newAiScore, humanScore)
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
		flag := 0
		if maxEval <= originalAlpha {
			flag = 2
		} else if maxEval >= beta {
			flag = 1
		}
		GlobalCache[cacheIdx] = CacheEntry{Hash: currentHash, Depth: depth, Value: maxEval, Flag: flag}
		return maxEval
	} else {
		minEval := math.MaxFloat64
		for _, move := range validMoves {
			var nextBoard []int
			var capCount int
			var nextHash uint64
			if move == -1 {
				nextBoard = make([]int, NumCells)
				copy(nextBoard, board)
				nextHash = currentHash
			} else {
				var legal bool
				nextBoard, capCount, legal = CheckMoveAndCapture(board, move, color)
				if !legal {
					continue
				}
				nextHash = currentHash ^ ZobristWhite[move]
			}
			newHumanScore := humanScore
			if color != aiColor {
				newHumanScore += capCount
			}

			eval := hybridAlphaBeta(nextHash, nextBoard, depth-1, alpha, beta, true, aiColor, -color, aiScore, newHumanScore)
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
		flag := 0
		if minEval <= originalAlpha {
			flag = 2
		} else if minEval >= beta {
			flag = 1
		}
		GlobalCache[cacheIdx] = CacheEntry{Hash: currentHash, Depth: depth, Value: minEval, Flag: flag}
		return minEval
	}
}

func GetHybridMove(weights []float64, board []int, color int, aiScore, humanScore int) (int, int, []int) {
	startTime := time.Now()
	initZobrist()
	EvaluatedNodes = 0

	bestCandidates := getOrderedMoves(board, color, 16)
	if len(bestCandidates) == 0 {
		return -1, 0, board
	}

	baseHash := computeZobristHash(board, aiScore, humanScore)

	type ScoredCandidate struct {
		Idx   int
		Value float64
	}
	var scoredList []ScoredCandidate

	bestMove := bestCandidates[0] // ФИКС ТИПОВ: Извлекаем первый int
	bestValue := -math.MaxFloat64
	alpha := -math.MaxFloat64
	beta := math.MaxFloat64

	for _, moveIdx := range bestCandidates {
		if time.Since(startTime) > 45*time.Millisecond {
			break
		}

		nextBoard, capCount, _ := CheckMoveAndCapture(board, moveIdx, color)
		if aiScore+capCount >= TargetScore {
			return moveIdx, capCount, nextBoard
		}

		nextHash := baseHash ^ ZobristBlack[moveIdx]
		boardValue := hybridAlphaBeta(nextHash, nextBoard, 7, alpha, beta, false, color, -color, aiScore+capCount, humanScore)

		scoredList = append(scoredList, ScoredCandidate{
			Idx:   moveIdx,
			Value: boardValue,
		})

		if boardValue > bestValue {
			bestValue = boardValue
			bestMove = moveIdx
		}
		if boardValue > alpha {
			alpha = boardValue
		}
	}

	var topGroup []int
	for _, cand := range scoredList {
		if math.Abs(cand.Value-bestValue) < 0.0001 {
			topGroup = append(topGroup, cand.Idx)
		}
	}
	if len(topGroup) > 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		bestMove = topGroup[r.Intn(len(topGroup))]
	}

	finalBoard, finalCap, _ := CheckMoveAndCapture(board, bestMove, color)
	return bestMove, finalCap, finalBoard
}
