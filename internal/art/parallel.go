package art

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Parallel runs f(0..n-1) on all CPU cores.
func Parallel(n int, f func(i int)) {
	var next atomic.Int64
	var wg sync.WaitGroup
	for k := 0; k < runtime.NumCPU(); k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= n {
					return
				}
				f(i)
			}
		}()
	}
	wg.Wait()
}
