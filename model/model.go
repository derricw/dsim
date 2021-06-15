package model

import (
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
)

type Pool chan time.Time

type ProcessReport struct {
	BatchesCompleted  int
	BatchesInProgress int
	Name              string
	CurrentBatch      map[string]int // not used yet
	IdleTime          time.Duration
}

type Process struct {
	// the channels that we can draw input from
	// and how many we need from each to do work
	in map[Pool]int
	// how long it takes to do the work
	duration time.Duration
	// how many output tokens do we create for each
	// unit of work
	out map[Pool]int
	// when did we last start a batch
	clock []time.Time
	// how long do we spend waiting on a batch to start
	idleTime time.Duration
	// how many batches have we completed and started
	batching          []int // not implemented
	startedBatches    int
	processingBatches int
	completedBatches  int
	// name so we can keep track of it
	name string
	// number of copies of this process
	replicas int
}

// Run runs the process until the specified duration has ended
func (p *Process) RunUntilTime(t time.Duration) {
	//log.Debugf("%s length: %d - %d", p.name, len(p.in), len(p.out))
	p.clock = make([]time.Time, p.replicas)
	t0 := time.Now()
	for c := range p.clock {
		p.clock[c] = t0
	}
	for {
		for r := 0; r < p.replicas; r++ {
			if p.in != nil && len(p.in) > 0 {
				last := p.consume(r)
				if last.After(p.clock[r]) {
					p.clock[r] = last
				}
			}
			finishTime := p.clock[r].Add(p.duration)
			if finishTime.Sub(t0) > t {
				return
			}
			p.clock[r] = finishTime
			p.produce(r)
		}
	}
}

// consume blocks until a replica consumes a batch
func (p *Process) consume(replica int) time.Time {
	log.Debugf("%s-%d started batch #%d @ %v", p.name, replica, p.startedBatches, p.clock[replica])
	p.startedBatches++
	var last time.Time = p.clock[replica]
	p.batching = make([]int, len(p.in)) // to keep track of how many we've batched (for reporting)
	cursor := 0
	for ch, c := range p.in {
		for i := 0; i < c; i++ {
			t := <-ch
			if last.Before(t) {
				last = t
			}
			p.batching[cursor]++
		}
		cursor++
	}
	// calculate idle time
	// idleTime = idleTime + (last - p.clock)
	if last.After(p.clock[replica]) {
		waitedFor := last.Sub(p.clock[replica])
		p.idleTime += waitedFor
	}

	log.Debugf("%s-%d processing batch #%d @ %v", p.name, replica, p.processingBatches, last)
	p.processingBatches++
	return last
}

func (p *Process) produce(replica int) {
	for ch, c := range p.out {
		for i := 0; i < c; i++ {
			ch <- p.clock[replica]
			//time.Sleep(1 * time.Millisecond)
		}
	}
	log.Debugf("%s-%d finished batch #%d @ %v", p.name, replica, p.completedBatches, p.clock[replica])
	p.completedBatches++
}

func (p *Process) Report() *ProcessReport {
	report := &ProcessReport{
		Name:              p.name,
		BatchesCompleted:  p.completedBatches,
		BatchesInProgress: p.completedBatches - p.startedBatches,
		IdleTime:          p.idleTime,
	}
	return report
}

func NewProcess(in, out map[Pool]int, duration time.Duration, name string, replicas int) *Process {
	p := &Process{
		in:       in,
		out:      out,
		duration: duration,
		name:     name,
		replicas: replicas,
	}
	return p
}

type ProcessConfig struct {
	In       map[string]int `yaml:"in"`
	Out      map[string]int `yaml:"out"`
	Duration time.Duration  `yaml:"duration"`
	Replicas int            `yaml:"replicas"`
}

type ModelConfig struct {
	Processes   map[string]ProcessConfig `yaml:"processes"`
	MaxPoolSize map[string]int           `yaml:"max_pool_size"`
}

func NewModelConfigFromFile(path string) (*ModelConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &ModelConfig{}
	err = yaml.Unmarshal(b, m)
	return m, nil
}

type Model struct {
	processes []*Process
	pools     map[string]Pool
}

func (m *Model) RunUntilTime(t time.Duration) {
	for _, p := range m.processes {
		go p.RunUntilTime(t)
	}
}

// Report prints a report on the state of our model.
// Run it after your conditions are met, or it won't be very
// interesting.
func (m *Model) Report() {
	// report what is in the pools
	poolNames := []string{}
	for name, _ := range m.pools {
		poolNames = append(poolNames, name)
	}
	sort.Strings(poolNames)

	for _, poolName := range poolNames {
		fmt.Printf("%s\t%d\n", poolName, len(m.pools[poolName]))
	}
}

func NewModelFromConfig(config *ModelConfig) (*Model, error) {
	processes := []*Process{}
	pools := buildPoolsFromConfig(config)
	for name, conf := range config.Processes {
		if conf.Replicas <= 0 {
			conf.Replicas = 1
		}
		in := map[Pool]int{}
		out := map[Pool]int{}

		for poolName, count := range conf.In {
			pool := pools[poolName]
			in[pool] = count
		}
		for poolName, count := range conf.Out {
			pool := pools[poolName]
			out[pool] = count
		}

		p := NewProcess(in, out, conf.Duration, name, conf.Replicas)
		processes = append(processes, p)
	}
	return &Model{
		processes: processes,
		pools:     pools,
	}, nil

}

func NewModelFromFile(path string) (*Model, error) {
	config, err := NewModelConfigFromFile(path)
	if err != nil {
		return nil, err
	}
	return NewModelFromConfig(config)
}

func buildPoolsFromConfig(config *ModelConfig) map[string]Pool {
	pools := map[string]Pool{}
	// add all pools with a pool config defined
	for poolName, maxSize := range config.MaxPoolSize {
		if pool := pools[poolName]; pool == nil {
			pools[poolName] = make(chan time.Time, maxSize)
		}
	}

	// add all pools from all processes
	for _, conf := range config.Processes {
		for poolName, _ := range conf.In {
			if pool := pools[poolName]; pool == nil {
				pools[poolName] = make(chan time.Time, 1000)
			}
		}
		for poolName, _ := range conf.Out {
			if pool := pools[poolName]; pool == nil {
				pools[poolName] = make(chan time.Time, 1000)
			}
		}
	}
	return pools
}
