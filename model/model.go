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
	clock time.Time
	// how long do we spend waiting on a batch to start
	idleTime time.Duration
	// how many batches have we completed and started
	batching          []int // not implemented
	startedBatches    int
	processingBatches int
	completedBatches  int
	// name so we can keep track of it
	name string
}

// Run runs the process until the specified duration has ended
func (p *Process) Run(until time.Duration) {
	//log.Debugf("%s length: %d - %d", p.name, len(p.in), len(p.out))
	t0 := time.Now()
	p.clock = t0
	if p.in == nil || len(p.in) == 0 {
		// we are a producer, first batch at t0
		// we might not want this?
		p.produce()
	}
	for {
		if p.in != nil && len(p.in) > 0 {
			last := p.consume()
			if last.After(p.clock) {
				p.clock = last
			}
		}
		finishTime := p.clock.Add(p.duration)
		if finishTime.Sub(t0) > until {
			return
		}
		p.clock = finishTime
		p.produce()
	}
}

func (p *Process) consume() time.Time {
	log.Debugf("%s started a batch @ %v", p.name, p.clock)
	p.startedBatches++
	var last time.Time
	for ch, c := range p.in {
		for i := 0; i < c; i++ {
			t := <-ch
			if last.Before(t) {
				last = t
			}
		}
	}
	// calculate idle time
	// idleTime = idleTime + (last - p.clock)
	if last.After(p.clock) {
		waitedFor := last.Sub(p.clock)
		p.idleTime += waitedFor
	}

	p.processingBatches++
	log.Debugf("%s processing a batch @ %v", p.name, last)
	return last
}

func (p *Process) produce() {
	for ch, c := range p.out {
		for i := 0; i < c; i++ {
			ch <- p.clock
			time.Sleep(1 * time.Millisecond)
		}
	}
	p.completedBatches++
	log.Debugf("%s finished a batch @ %v", p.name, p.clock)
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

func NewProcess(in, out map[Pool]int, duration time.Duration, name string) *Process {
	p := &Process{
		in:       in,
		out:      out,
		duration: duration,
		name:     name,
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

func (m *Model) Run(until time.Duration) {
	for _, p := range m.processes {
		go p.Run(until)
	}
}

// Report prints a report on the state of our model.
// Run it after your conditions are met, or it won't be very
// interesting.
func (m *Model) Report() {
	poolNames := []string{}
	for name, _ := range m.pools {
		poolNames = append(poolNames, name)
	}
	sort.Strings(poolNames)

	// sort this!
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

		// TODO: this got ugly as shit holy god
		for i := 0; i < conf.Replicas; i++ {
			log.Debugf("%s: %v", name, conf.In)
			for poolName, count := range conf.In {
				log.Debugf("%s: %s %d", name, poolName, count)
				pool := pools[poolName]
				in[pool] = count
			}
			log.Debugf("%s: %v", name, conf.Out)
			for poolName, count := range conf.Out {
				log.Debugf("%s: %s %d", name, poolName, count)
				pool := pools[poolName]
				out[pool] = count
			}

			replicaName := fmt.Sprintf("%s-%d", name, i)

			p := NewProcess(in, out, conf.Duration, replicaName)
			processes = append(processes, p)
		}
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
