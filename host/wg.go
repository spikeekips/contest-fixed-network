package host

import (
	"sync"
)

func RunWaitGroup(n int, callback func(int) error) error {
	var wg sync.WaitGroup
	wg.Add(n)

	errChan := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()

			errChan <- callback(i)
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
