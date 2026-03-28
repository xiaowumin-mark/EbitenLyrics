package bgrender

// 文件说明：背景模块内部的随机数辅助函数。
// 主要职责：集中管理随机源，避免各处重复创建。

import (
	"math/rand"
	"sync"
	"time"
)

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func randFloat64() float64 {
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Float64()
}

func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Intn(n)
}
