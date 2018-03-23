// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tgo

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tmath"
)

const (
	// MetricProcessStart is the metric name storing the time when this process
	// has been started.
	MetricProcessStart = "ProcessStart"
	// MetricGoRoutines is the metric name storing the number of active go
	// routines.
	MetricGoRoutines = "GoRoutines"
	// MetricGoVersion holds the go version as Major*10000+Minor*100+Patch
	MetricGoVersion = "GoVersion"
	// MetricMemoryAllocated holds the currently active memory in bytes
	MetricMemoryAllocated = "GoMemoryAllocated"
	// MetricMemoryNumObjects holds the total number of allocated heap objects
	MetricMemoryNumObjects = "GoMemoryNumObjects"
	// MetricMemoryGCEnabled holds 1 or 0 depending on the state of garbage collection
	MetricMemoryGCEnabled = "GoMemoryGCEnabled"
)

// ProcessStartTime stores the time this process has started.
// This value is also stored in the metric MetricProcessStart
var ProcessStartTime time.Time

// Metric allows any part of gollum to store and/or modify metric values by
// name.
var Metric = (*Metrics)(nil)

// Metrics is the container struct for runtime metrics that can be used with
// the metrics server.
type Metrics struct {
	store      map[string]*int64
	rates      map[string]*rate
	storeGuard *sync.RWMutex
	rateGuard  *sync.RWMutex
}

type rate struct {
	metric     string
	samples    tcontainer.Int64Slice
	ticker     *time.Ticker
	lastSample int64
	value      int64
	index      uint64
	numMedians int
	relative   bool
}

func init() {
	ProcessStartTime = time.Now()
}

// EnableGlobalMetrics initializes the Metric global variable (if it is not nil)
// This function is not threadsafe and should be called once directly after the
// process started.
func EnableGlobalMetrics() {
	if Metric == nil {
		Metric = NewMetrics()
	}
}

// NewMetrics creates a new metrics container.
// To initialize the global Metrics variable use EnableGlobalMetrics.
func NewMetrics() *Metrics {
	return &Metrics{
		store:      make(map[string]*int64),
		rates:      make(map[string]*rate),
		storeGuard: new(sync.RWMutex),
		rateGuard:  new(sync.RWMutex),
	}
}

// InitSystemMetrics Adds system metrics (memory, go routines, etc.) to this
// metric storage. System metrics need to be updated manually by
// calling UpdateSystemMetrics().
func (met *Metrics) InitSystemMetrics() {
	met.New(MetricProcessStart)
	met.New(MetricGoRoutines)
	met.New(MetricGoVersion)
	met.New(MetricMemoryAllocated)
	met.New(MetricMemoryNumObjects)
	met.New(MetricMemoryGCEnabled)
	met.Set(MetricProcessStart, ProcessStartTime.Unix())

	version := runtime.Version()
	if version[0] == 'g' && version[1] == 'o' {
		parts := strings.Split(version[2:], ".")
		numericVersion := make([]uint64, tmath.MaxI(3, len(parts)))
		for i, p := range parts {
			numericVersion[i], _ = strconv.ParseUint(p, 10, 64)
		}

		met.SetI(MetricGoVersion, int(numericVersion[0]*10000+numericVersion[1]*100+numericVersion[2]))
	}

	met.UpdateSystemMetrics()
}

// Close stops the internal go routines used for e.g. sampling
func (met *Metrics) Close() {
	for _, r := range met.rates {
		r.ticker.Stop()
	}
}

// New creates a new metric under the given name with a value of 0
func (met *Metrics) New(name string) {
	met.new(name)
}

// NewRate creates a new rate. Rates are based on another metric and sample
// this given base metric every second. When numSamples have been stored, old
// samples are overriden (oldest first).
// Retrieving samples via GetRate will calculate the median of a set of means.
// numMedianSamples defines how many values will be used for mean calculation.
// A value of 0 will calculate the mean value of all samples. A value of 1 or
// a value >= numSamples will build a median over all samples. Any other
// value will divide the stored samples into the given number of groups and
// build a median over the mean of all these groups.
// The relative parameter defines if the samples are taking by storing the
// current value (false) or the difference to the last sample (true).
func (met *Metrics) NewRate(baseMetric string, name string, interval time.Duration, numSamples uint8, numMedianSamples uint8, relative bool) error {
	met.storeGuard.RLock()
	if _, exists := met.store[baseMetric]; !exists {
		met.storeGuard.RUnlock()
		return fmt.Errorf("Metric %s is not registered", baseMetric)
	}
	met.storeGuard.RUnlock()

	met.rateGuard.Lock()
	defer met.rateGuard.Unlock()

	if _, exists := met.rates[name]; exists {
		return fmt.Errorf("Rate %s is already registered", name)
	}

	if numMedianSamples >= numSamples {
		numMedianSamples = 0
	}

	newRate := &rate{
		metric:     baseMetric,
		samples:    make(tcontainer.Int64Slice, numSamples),
		numMedians: int(numMedianSamples),
		lastSample: 0,
		value:      0,
		index:      0,
		ticker:     time.NewTicker(interval),
		relative:   relative,
	}

	met.rates[name] = newRate

	go func() {
		isRunning := true
		for isRunning {
			_, isRunning = <-newRate.ticker.C
			met.updateRate(newRate)
		}
	}()

	return nil
}

// Set sets a given metric to a given value.
func (met *Metrics) Set(name string, value int64) {
	atomic.StoreInt64(met.get(name), value)
}

// SetI is Set for int values (conversion to int64)
func (met *Metrics) SetI(name string, value int) {
	atomic.StoreInt64(met.get(name), int64(value))
}

// SetF is Set for float64 values (conversion to int64)
func (met *Metrics) SetF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.StoreInt64(met.get(name), int64(rounded))
}

// SetB is Set for boolean values (conversion to 0/1)
func (met *Metrics) SetB(name string, value bool) {
	if value {
		atomic.StoreInt64(met.get(name), int64(1))
	} else {
		atomic.StoreInt64(met.get(name), int64(0))
	}
}

// Inc adds 1 to a given metric.
func (met *Metrics) Inc(name string) {
	atomic.AddInt64(met.get(name), 1)
}

// Dec subtracts 1 from a given metric.
func (met *Metrics) Dec(name string) {
	atomic.AddInt64(met.get(name), -1)
}

// Add adds a number to a given metric.
func (met *Metrics) Add(name string, value int64) {
	atomic.AddInt64(met.get(name), value)
}

// AddI is Add for int values (conversion to int64)
func (met *Metrics) AddI(name string, value int) {
	atomic.AddInt64(met.get(name), int64(value))
}

// AddF is Add for float64 values (conversion to int64)
func (met *Metrics) AddF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.AddInt64(met.get(name), int64(rounded))
}

// Sub subtracts a number to a given metric.
func (met *Metrics) Sub(name string, value int64) {
	atomic.AddInt64(met.get(name), -value)
}

// SubI is SubI for int values (conversion to int64)
func (met *Metrics) SubI(name string, value int) {
	atomic.AddInt64(met.get(name), int64(-value))
}

// SubF is Sub for float64 values (conversion to int64)
func (met *Metrics) SubF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.AddInt64(met.get(name), int64(-rounded))
}

// Get returns the value of a given metric or rate.
// If the value does not exists error is non-nil and the returned value is 0.
func (met *Metrics) Get(name string) (int64, error) {
	if metric := met.tryGetMetric(name); metric != nil {
		return atomic.LoadInt64(metric), nil
	}

	if rate := met.tryGetRate(name); rate != nil {
		return *rate, nil
	}

	// Neither rate nor metric found
	return 0, fmt.Errorf("Metric %s not found", name)
}

// Dump creates a JSON string from all stored metrics.
func (met *Metrics) Dump() ([]byte, error) {
	snapshot := make(map[string]int64)

	met.storeGuard.RLock()
	for key, value := range met.store {
		snapshot[key] = atomic.LoadInt64(value)
	}
	met.storeGuard.RUnlock()

	met.rateGuard.RLock()
	for key, rate := range met.rates {
		snapshot[key] = rate.value
	}
	met.rateGuard.RUnlock()

	return json.Marshal(snapshot)
}

// ResetMetrics resets all registered key values to 0 expect for system Metrics.
// This locks all writes in the process.
func (met *Metrics) ResetMetrics() {
	met.storeGuard.Lock()
	for key := range met.store {
		switch key {
		case MetricProcessStart, MetricGoRoutines, MetricGoVersion:
			// ignore
		default:
			*met.store[key] = 0
		}
	}
	met.storeGuard.Unlock()

	met.rateGuard.Lock()
	for _, rate := range met.rates {
		rate.lastSample = 0
		rate.value = 0
		rate.index = 0
		rate.samples.Set(0)
	}
	met.rateGuard.Unlock()
}

// FetchAndReset resets all of the given keys to 0 and returns the
// value before the reset as array. If a given metric does not exist
// it is ignored. This locks all writes in the process.
func (met *Metrics) FetchAndReset(keys ...string) map[string]int64 {
	state := make(map[string]int64)

	met.storeGuard.Lock()
	for _, key := range keys {
		if val, exists := met.store[key]; exists {
			state[key] = *val
			*val = 0
		}
	}
	met.storeGuard.Unlock()

	met.rateGuard.Lock()
	for _, key := range keys {
		if rate, exists := met.rates[key]; exists {
			rate.lastSample = 0
			rate.value = 0
			rate.index = 0
			rate.samples.Set(0)
		}
	}
	met.rateGuard.Unlock()

	return state
}

// UpdateSystemMetrics updates all default or system based metrics like memory
// consumption and number of go routines. This function is not called
// automatically.
func (met *Metrics) UpdateSystemMetrics() {
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)

	met.SetI(MetricGoRoutines, runtime.NumGoroutine())
	met.Set(MetricMemoryAllocated, int64(stats.Alloc))
	met.SetB(MetricMemoryGCEnabled, stats.EnableGC)
	met.Set(MetricMemoryNumObjects, int64(stats.HeapObjects))
}

func (met *Metrics) updateRate(r *rate) {
	met.rateGuard.RLock()
	defer met.rateGuard.RUnlock()

	// Read current values in a snapshot to avoid deadlocks
	sample := atomic.LoadInt64(met.get(r.metric))
	idx := r.index % uint64(len(r.samples))
	r.index++

	// Sample metric
	if r.relative {
		r.samples[idx] = sample - r.lastSample
		r.lastSample = sample
	} else {
		r.samples[idx] = sample
	}

	numSamples := int64(tmath.MinUint64(idx+1, uint64(len(r.samples))))

	// Build value
	switch {
	case r.numMedians == 1:
		// Mean of all values
		total := int64(0)
		for _, s := range r.samples[:numSamples] {
			total += s
		}
		r.value = total / numSamples

	case r.numMedians == 0 || numSamples <= int64(r.numMedians):
		// Median of all values
		values := make(tcontainer.Int64Slice, numSamples)
		copy(values, r.samples[:numSamples])
		values.Sort()
		r.value = values[numSamples/2]

	default:
		// Median of means
		blockSize := float32(numSamples) / float32(r.numMedians)
		blocks := make(tcontainer.Float32Slice, r.numMedians)

		for i, s := range r.samples[:numSamples] {
			blockIdx := int(float32(i) / blockSize)
			blocks[blockIdx] += float32(s)
		}

		blocks.Sort()
		r.value = int64(blocks[r.numMedians/2] / blockSize)
	}
}

func (met *Metrics) new(name string) *int64 {
	met.storeGuard.Lock()
	value, exists := met.store[name]
	if !exists {
		value = new(int64)
		met.store[name] = value
	}
	met.storeGuard.Unlock()

	return value
}

func (met *Metrics) get(name string) *int64 {
	met.storeGuard.RLock()
	v, exists := met.store[name]
	met.storeGuard.RUnlock()

	if exists {
		return v // ### return, exists ###
	}
	return met.new(name)
}

func (met *Metrics) tryGetMetric(name string) *int64 {
	met.storeGuard.RLock()
	v, exists := met.store[name]
	met.storeGuard.RUnlock()

	if exists {
		return v // ### return, exists ###
	}
	return nil
}

func (met *Metrics) tryGetRate(name string) *int64 {
	met.rateGuard.RLock()
	r, exists := met.rates[name]
	met.rateGuard.RUnlock()

	if exists {
		return &r.value // ### return, exists ###
	}
	return nil
}
