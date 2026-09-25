package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"

	"atari/engine"
)

const (
	WeightFile = "weights.bin"
	PopSize    = 12
	MaxGen     = 30
	LogPeriod  = 2
	TestGames  = 25
)

func main() {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	alphaAI := NewAgent()

	if data, err := os.ReadFile(WeightFile); err == nil {
		if alphaAI.FromBytes(data) {
			fmt.Printf("⏳ Найдено бинарное сохранение CNN (%s). Веса восстановлены бит-в-бит.\n", WeightFile)
		} else {
			fmt.Println("⚠️ Файл weights.bin поврежден. Начинаем с нуля.")
		}
	} else {
		fmt.Println("🌱 Файл весов не найден. Начинаем эволюцию с чистого листа...")
	}

	currentAlphaWinRate := EvaluateAgainstBenchmark(alphaAI, TestGames)

	fmt.Printf("🔥 Запуск CNN эволюции до %d камней на %d ядрах CPU...\n", engine.TargetScore, numCPUs)
	fmt.Printf("📊 Стартовый рейтинг Альфы: %.1f%%\n", currentAlphaWinRate)

	maxPossibleScore := (PopSize - 1) * 2 * 2

	for gen := 1; gen <= MaxGen; gen++ {
		population := make([]*Agent, PopSize)
		// ФИКС: Кладём копию босса строго в нулевой индекс слайса
		population[0] = alphaAI.Clone()

		for i := 1; i < PopSize; i++ {
			mutant := alphaAI.Clone()
			Mutate(mutant, 0.15, 0.06)
			population[i] = mutant
		}

		type Match struct{ p1Idx, p2Idx int }
		var matches []Match
		for i := 0; i < PopSize; i++ {
			for j := i + 1; j < PopSize; j++ {
				matches = append(matches, Match{p1Idx: i, p2Idx: j})
			}
		}

		var wg sync.WaitGroup
		matchChan := make(chan Match, len(matches))
		var scoreMutex sync.Mutex

		for w := 0; w < numCPUs; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for match := range matchChan {
					p1 := population[match.p1Idx]
					p2 := population[match.p2Idx]

					res1, t1 := PlayMatch(p1, p2)
					res2, t2 := PlayMatch(p2, p1)

					p1Points, p2Points := 0, 0

					if res1 == 1 {
						p1Points += 2
					} else if res1 == -1 {
						p2Points += 2
					} else {
						p1Points++
						p2Points++
					}
					if res2 == 1 {
						p2Points += 2
					} else if res2 == -1 {
						p1Points += 2
					} else {
						p1Points++
						p2Points++
					}

					scoreMutex.Lock()
					population[match.p1Idx].Score += p1Points
					population[match.p2Idx].Score += p2Points
					population[match.p1Idx].AvgTurns += float64(t1)
					population[match.p2Idx].AvgTurns += float64(t2)
					scoreMutex.Unlock()
				}
			}()
		}
		for _, m := range matches {
			matchChan <- m
		}
		close(matchChan)
		wg.Wait()

		bestIdx := 0
		maxScore := -1
		minTurns := math.MaxFloat64
		for i := 0; i < PopSize; i++ {
			if population[i].Score > maxScore {
				maxScore = population[i].Score
				minTurns = population[i].AvgTurns
				bestIdx = i
			} else if population[i].Score == maxScore && population[i].AvgTurns < minTurns {
				minTurns = population[i].AvgTurns
				bestIdx = i
			}
		}

		alphaUpgraded := 0
		champion := population[bestIdx]
		testCandidate := champion.Clone()

		mutantWinRate := EvaluateAgainstBenchmark(testCandidate, TestGames)

		if mutantWinRate >= currentAlphaWinRate {
			copy(alphaAI.Weights, testCandidate.Weights)
			currentAlphaWinRate = mutantWinRate
			alphaUpgraded = 1
		}

		if gen%LogPeriod == 0 || gen == 1 {
			fmt.Printf("Поколение %d/%d | Топ-счет: %d/%d | Ходы: %.1f | Альфа обновлен: %d | Истинный WinRate: %.1f%%\n", gen, MaxGen, maxScore, maxPossibleScore, minTurns/2.0, alphaUpgraded, currentAlphaWinRate)
		}

		_ = os.WriteFile(WeightFile, alphaAI.ToBytes(), 0644)
	}
}
