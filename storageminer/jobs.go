package storageminer

import (
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func (m *StorageMiner) jobsRecords() error {
	wg := sync.WaitGroup{}
	wg.Add(len(m.miners))

	for maddr := range m.miners {
		go func(maddr address.Address) {
			defer wg.Done()
			err := m.jobsRecord(maddr)
			if err != nil {
				log.Errorw("jobsRecord failed", "miner", maddr, "err", err)
			} else {
				log.Infow("jobsRecord success", "miner", maddr)
			}
		}(maddr)
	}
	wg.Wait()
	return nil
}

func (m *StorageMiner) jobsRecord(maddr address.Address) error {
	ctx, _ := tag.New(m.ctx,
		tag.Upsert(metrics.MinerID, maddr.String()),
	)
	api := m.miners[maddr].api
	size := m.miners[maddr].size

	jobss, err := api.WorkerJobs(ctx)
	if err != nil {
		return err
	}

	result := map[string]int64{}
	for _, jobs := range jobss {
		for _, job := range jobs {
			if job.RunWait != 0 {
				continue
			}
			td, ok := m.running[size][job.Task.Short()]
			if ok && time.Since(job.Start) > td {
				result[job.Task.Short()] += 1
			}
		}
	}

	for task, n := range result {
		ctx, _ = tag.New(ctx,
			tag.Upsert(metrics.TaskType, task),
		)
		stats.Record(ctx, metrics.JobsTimeout.M(n))
	}

	return nil
}
