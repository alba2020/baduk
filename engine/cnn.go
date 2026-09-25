package engine

func ForwardCNN(weights []float64, board []float64) []float64 {
	w1Start := 0
	w1End := NumFilters * 3 * 3
	b1Start := w1End
	b1End := b1Start + NumFilters
	w2Start := b1End
	w2End := w2Start + (NumFilters * 3 * 3 * NumFilters)
	b2Start := w2End
	b2End := b2Start + NumFilters
	w3Start := b2End
	w3End := w3Start + NumFilters
	b3Idx := w3End

	fMaps1 := make([]float64, BoardSize*BoardSize*NumFilters)
	for f := 0; f < NumFilters; f++ {
		fIdx := f * NumCells
		wIdx := w1Start + f*9
		bias := weights[b1Start+f]
		for r := 0; r < BoardSize; r++ {
			for c := 0; c < BoardSize; c++ {
				sum := bias
				for ky := -1; ky <= 1; ky++ {
					for kx := -1; kx <= 1; kx++ {
						nr, nc := r+ky, c+kx
						if nr >= 0 && nr < BoardSize && nc >= 0 && nc < BoardSize {
							sum += board[nr*BoardSize+nc] * weights[wIdx+(ky+1)*3+(kx+1)]
						}
					}
				}
				if sum < 0 {
					sum *= 0.01
				}
				fMaps1[fIdx+r*BoardSize+c] = sum
			}
		}
	}

	fMaps2 := make([]float64, BoardSize*BoardSize*NumFilters)
	for fOut := 0; fOut < NumFilters; fOut++ {
		fOutIdx := fOut * NumCells
		bias := weights[b2Start+fOut]
		for r := 0; r < BoardSize; r++ {
			for c := 0; c < BoardSize; c++ {
				sum := bias
				for ky := -1; ky <= 1; ky++ {
					for kx := -1; kx <= 1; kx++ {
						nr, nc := r+ky, c+kx
						if nr >= 0 && nr < BoardSize && nc >= 0 && nc < BoardSize {
							for fIn := 0; fIn < NumFilters; fIn++ {
								wIdx := w2Start + (fOut*NumFilters+fIn)*9
								sum += fMaps1[fIn*NumCells+nr*BoardSize+nc] * weights[wIdx+(ky+1)*3+(kx+1)]
							}
						}
					}
				}
				if sum < 0 {
					sum *= 0.01
				}
				fMaps2[fOutIdx+r*BoardSize+c] = sum
			}
		}
	}

	output := make([]float64, NumCells)
	finalBias := weights[b3Idx]
	for i := 0; i < NumCells; i++ {
		sum := finalBias
		for f := 0; f < NumFilters; f++ {
			sum += fMaps2[f*NumCells+i] * weights[w3Start+f]
		}
		output[i] = sum
	}
	return output
}
