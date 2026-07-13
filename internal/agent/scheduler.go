package agent

import (
	"context"
	"sync"
)

type Scheduler struct {
	MaxParallel int
}

func (s Scheduler) Run(ctx context.Context, projectIDs []string, job func(context.Context, string)) {
	maximum := s.MaxParallel
	if maximum < 1 {
		maximum = 1
	}
	unique := make([]string, 0, len(projectIDs))
	seen := make(map[string]struct{}, len(projectIDs))
	for _, projectID := range projectIDs {
		if _, exists := seen[projectID]; exists {
			continue
		}
		seen[projectID] = struct{}{}
		unique = append(unique, projectID)
	}
	queue := make(chan string)
	var workers sync.WaitGroup
	for index := 0; index < maximum; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for projectID := range queue {
				if ctx.Err() != nil {
					continue
				}
				job(ctx, projectID)
			}
		}()
	}
	for _, projectID := range unique {
		select {
		case queue <- projectID:
		case <-ctx.Done():
			close(queue)
			workers.Wait()
			return
		}
	}
	close(queue)
	workers.Wait()
}
