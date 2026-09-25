package engine

const (
	BoardSize   = 9
	NumCells    = BoardSize * BoardSize
	NumFilters  = 8
	TotalParam  = (NumFilters * 3 * 3) + NumFilters + (NumFilters * 3 * 3 * NumFilters) + NumFilters + (1 * 1 * NumFilters) + 1
	TargetScore = 5
)

type ScoredMove struct {
	Idx   int
	Logit float64
}
