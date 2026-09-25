package engine

func getGroupStats(board []int, startIdx int, color int) (group []int, liberties int) {
	visited := make([]bool, NumCells)
	countedLibs := make([]bool, NumCells)
	queue := []int{startIdx}
	visited[startIdx] = true

	head := 0
	for head < len(queue) {
		curr := queue[head]
		head++
		group = append(group, curr)

		x, y := curr/BoardSize, curr%BoardSize
		dx := []int{-1, 1, 0, 0}
		dy := []int{0, 0, -1, 1}

		for i := 0; i < 4; i++ {
			nx, ny := x+dx[i], y+dy[i]
			if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
				nIdx := nx*BoardSize + ny
				if board[nIdx] == 0 {
					if !countedLibs[nIdx] {
						liberties++
						countedLibs[nIdx] = true
					}
				} else if board[nIdx] == color && !visited[nIdx] {
					visited[nIdx] = true
					queue = append(queue, nIdx)
				}
			}
		}
	}
	return group, liberties
}

func CheckMoveAndCapture(currentBoard []int, moveIdx int, color int) ([]int, int, bool) {
	if moveIdx < 0 || moveIdx >= NumCells || currentBoard[moveIdx] != 0 {
		return nil, 0, false
	}

	nextBoard := make([]int, NumCells)
	copy(nextBoard, currentBoard)
	nextBoard[moveIdx] = color

	opponent := -color
	capturedStones := 0
	visitedOpponents := make([]bool, NumCells)

	// Temporary slice to hold stones slated for erasure
	var stonesToRemove []int

	// Step 1: Scan entire board for dead opponent groups atomically
	for i := 0; i < NumCells; i++ {
		if nextBoard[i] == opponent && !visitedOpponents[i] {
			oppGroup, liberties := getGroupStats(nextBoard, i, opponent)
			for _, member := range oppGroup {
				visitedOpponents[member] = true
			}

			if liberties == 0 {
				stonesToRemove = append(stonesToRemove, oppGroup...)
			}
		}
	}

	// Step 2: Clear captured stones all at once
	capturedStones = len(stonesToRemove)
	for _, idx := range stonesToRemove {
		nextBoard[idx] = 0
	}

	// Step 3: Check for suicide logic post-capture
	_, liberties := getGroupStats(nextBoard, moveIdx, color)
	if liberties == 0 && capturedStones == 0 {
		return nil, 0, false
	}

	return nextBoard, capturedStones, true
}

func isCellInAtari(board []int, idx int, color int) bool {
	if board[idx] != color {
		return false
	}
	_, liberties := getGroupStats(board, idx, color)
	return liberties == 1
}

func countTotalLiberties(board []int, color int) int {
	total := 0
	visited := make([]bool, NumCells)
	for i := 0; i < NumCells; i++ {
		if board[i] == color && !visited[i] {
			group, liberties := getGroupStats(board, i, color)
			for _, m := range group {
				visited[m] = true
			}
			total += liberties
		}
	}
	return total
}
