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
	"github.com/trivago/tgo/ttesting"
	"sync"
	"testing"
	"time"
)

func getMockMetric() *Metrics {
	return &Metrics{
		store:      make(map[string]*int64),
		rates:      make(map[string]*rate),
		storeGuard: new(sync.RWMutex),
		rateGuard:  new(sync.RWMutex),
	}
}

func TestRateAbsolute(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()
	defer mockMetric.Close()

	mockMetric.New("base")
	err := mockMetric.NewRate("invalid", "mean", time.Hour, 10, 0, false)
	expect.NotNil(err)

	err = mockMetric.NewRate("base", "median", time.Hour, 10, 0, false)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "mean", time.Hour, 10, 1, false)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "median3", time.Hour, 10, 3, false)
	expect.NoError(err)

	for i := 0; i < 10; i++ {
		if i < 5 {
			mockMetric.Set("base", 2)
		} else {
			mockMetric.Set("base", 4)
		}
		for _, r := range mockMetric.rates {
			mockMetric.updateRate(r)
		}
	}

	// values : 2  2  2  2  2  4  4  4  4  4
	// mean   : 30 / 10 = 3
	// median : 2  2  2  2  2 [4] 4  4  4  4
	// med3   :[   2      ][  >3<  ][   4   ]

	value, err := mockMetric.Get("mean")
	expect.NoError(err)
	expect.Equal(int64(3), value)

	value, err = mockMetric.Get("median")
	expect.NoError(err)
	expect.Equal(int64(4), value)

	value, err = mockMetric.Get("median3")
	expect.NoError(err)
	expect.Equal(int64(3), value)
}

func TestRateRelative(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()
	defer mockMetric.Close()

	mockMetric.New("base")
	err := mockMetric.NewRate("invalid", "mean", time.Hour, 9, 0, true)
	expect.NotNil(err)

	err = mockMetric.NewRate("base", "median", time.Hour, 9, 0, true)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "mean", time.Hour, 9, 1, true)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "median3", time.Hour, 9, 3, true)
	expect.NoError(err)

	for i := 1; i < 10; i++ {
		if i < 5 {
			mockMetric.Set("base", int64(i*2))
		} else {
			mockMetric.Set("base", int64(i*3))
		}
		for _, r := range mockMetric.rates {
			mockMetric.updateRate(r)
		}
	}

	// values : 2  4  6  8 15 18 21 24 27
	// diff   : 2  2  2  2  7  3  3  3  3
	// mean   : 27 / 9 = 3
	// median : 2  2  2  2 [3] 3  3  3  7
	// med3   :[   2   ][  >3<  ][   4   ]

	value, err := mockMetric.Get("mean")
	expect.NoError(err)
	expect.Equal(int64(3), value)

	value, err = mockMetric.Get("median")
	expect.NoError(err)
	expect.Equal(int64(3), value)

	value, err = mockMetric.Get("median3")
	expect.NoError(err)
	expect.Equal(int64(3), value)
}

func TestRateIncomplete(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()
	defer mockMetric.Close()

	mockMetric.New("base")
	err := mockMetric.NewRate("invalid", "mean", time.Hour, 10, 0, false)
	expect.NotNil(err)

	err = mockMetric.NewRate("base", "median", time.Hour, 10, 0, false)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "mean", time.Hour, 10, 1, false)
	expect.NoError(err)
	err = mockMetric.NewRate("base", "median3", time.Hour, 10, 3, false)
	expect.NoError(err)

	for i := 0; i < 5; i++ {
		if i < 2 {
			mockMetric.Set("base", 2)
		} else {
			mockMetric.Set("base", 4)
		}
		for _, r := range mockMetric.rates {
			mockMetric.updateRate(r)
		}
	}

	// values : 2  2  4  4  4  0  0  0  0  0
	// mean   : 16 / 5 = 3
	// median : 2  2 [4] 4  4  0  0  0  0  0
	// med3   :[  2 ][  4 ][2] -> 2 [2] 4

	value, err := mockMetric.Get("mean")
	expect.NoError(err)
	expect.Equal(int64(3), value)

	value, err = mockMetric.Get("median")
	expect.NoError(err)
	expect.Equal(int64(4), value)

	value, err = mockMetric.Get("median3")
	expect.NoError(err)
	expect.Equal(int64(2), value)
}

func TestMetricsSet(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()

	// test for initialization to zero
	mockMetric.New("MockMetric")
	count, err := mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(0), count)

	// test for setting to a particular value
	mockMetric.Set("MockMetric", int64(5))
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(5), count)

	// test for setting to a particular int
	mockMetric.SetI("MockMetric", 5)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(5), count)

	// test for setting to a particular float
	mockMetric.SetF("MockMetric", 4.3)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(4), count)

	// test for setting a particular boolean value
	mockMetric.SetB("MockMetric", true)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(1), count)

	mockMetric.SetB("MockMetric", false)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(0), count)
}

func TestMetricsAddSub(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()

	mockMetric.New("MockMetric")
	mockMetric.Add("MockMetric", int64(1))
	count, err := mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(1), count)

	mockMetric.AddI("MockMetric", 1)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(2), count)

	mockMetric.AddF("MockMetric", 2.4)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(4), count)

	mockMetric.Sub("MockMetric", int64(1))
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(3), count)

	mockMetric.SubF("MockMetric", 1.6)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(1), count)

	mockMetric.SubI("MockMetric", 1)
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(0), count)
}

func TestMetricsIncDec(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()
	mockMetric.New("MockMetric")

	mockMetric.Inc("MockMetric")
	count, err := mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(1), count)

	mockMetric.Dec("MockMetric")
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(0), count)

}

func TestMetricsReset(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()
	mockMetric.New("MockMetric")

	mockMetric.Set("MockMetric", int64(10))
	count, err := mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(10), count)

	mockMetric.ResetMetrics()
	count, err = mockMetric.Get("MockMetric")
	expect.NoError(err)
	expect.Equal(int64(0), count)
}

func TestFetchAndReset(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockMetric := getMockMetric()

	mockMetric.New("foo")
	mockMetric.Set("foo", int64(10))
	foo, err := mockMetric.Get("foo")
	expect.NoError(err)
	expect.Equal(int64(10), foo)

	mockMetric.New("bar")
	mockMetric.Set("bar", int64(20))
	bar, err := mockMetric.Get("bar")
	expect.NoError(err)
	expect.Equal(int64(20), bar)

	values := mockMetric.FetchAndReset("foo", "bar")
	expect.MapEqual(values, "foo", int64(10))
	expect.MapEqual(values, "bar", int64(20))

	foo, err = mockMetric.Get("foo")
	expect.NoError(err)
	expect.Equal(int64(0), foo)

	bar, err = mockMetric.Get("bar")
	expect.NoError(err)
	expect.Equal(int64(0), bar)
}
