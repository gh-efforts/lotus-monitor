package storageminer

import (
	"sync"
	"time"

	"github.com/gh-efforts/lotus-monitor/config"
	"github.com/gh-efforts/lotus-monitor/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func (m *StorageMiner) jobsRecords() {
	stop := metrics.Timer(m.ctx, "storageminer/jobsRecords")
	defer stop()

	miners := m.dc.MinersInfo()
	log.Debug(miners)

	wg := sync.WaitGroup{}
	wg.Add(len(miners))

	for _, mi := range miners {
		go func(mi config.MinerInfo) {
			defer wg.Done()
			if err := m.jobsRecord(mi); err != nil {
				log.Errorw("jobsRecord failed", "miner", mi.Address, "err", err)
				metrics.RecordError(m.ctx, "storageminer/jobsRecord")
			} else {
				log.Debugw("jobsRecord success", "miner", mi.Address)
			}
		}(mi)
	}
	wg.Wait()
}

func (m *StorageMiner) jobsRecord(mi config.MinerInfo) error {
	ctx, _ := tag.New(m.ctx,
		tag.Upsert(metrics.MinerID, mi.Address.String()),
	)
	api := mi.Api
	size := mi.Size

	jobss, err := api.WorkerJobs(ctx)
	if err != nil {
		return err
	}

	result := map[string]int64{}
	for task := range m.dc.Running[size] {
		result[task.Short()] = 0
	}
	for _, jobs := range jobss {
		for _, job := range jobs {
			if job.RunWait != 0 {
				continue
			}
			td, ok := m.dc.Running[size][job.Task]
			if ok && time.Since(job.Start) > time.Duration(td) {
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
