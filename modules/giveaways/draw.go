package giveaways

import "math/rand"

// drawWinners picks up to n distinct winners uniformly at random from entrants.
// It returns fewer than n (possibly zero) when there aren't enough participants.
func drawWinners(entrants []int64, n int) []int64 {
	if n <= 0 || len(entrants) == 0 {
		return nil
	}
	pool := make([]int64, len(entrants))
	copy(pool, entrants)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}
