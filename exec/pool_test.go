package exec

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	//"math/rand"
)

var ctx = context.Background()

type TestTask[T any] struct {
	result    T
	wgCount   *sync.WaitGroup
	wgLock    *sync.WaitGroup
	processed bool
}

func (t *TestTask[T]) Run() (T, error) {
	logger.InfoC(ctx, "Start executing task '%v'", t.result)
	//randTime := rand.Intn(2) + 1
	//time.Sleep(time.Duration(randTime)* time.Second)
	t.processed = true
	if t.wgCount != nil {
		t.wgCount.Done()
	}
	// wait to block all tasks which currently consume all executor goroutines, the amount must be equal to the parallelism value
	if t.wgLock != nil {
		t.wgLock.Wait()
	}
	return t.result, nil
}

type Result[T any] struct {
	err     error
	results []T
}

func TestPool(t *testing.T) {
	r := require.New(t)
	timeoutDuration := 5 * time.Minute
	parallelism := runtime.NumCPU()
	//parallelism := 2
	var expectedResult []int
	taskNumber := parallelism * 4
	var wgLock sync.WaitGroup
	wgLock.Add(1)
	var wgCount sync.WaitGroup
	wgCount.Add(parallelism)
	logger.InfoC(ctx, "Running test with CPU num=%d and tasks num=%d", parallelism, taskNumber)
	var tasks []Task[int]
	for i := 0; i < taskNumber; i++ {
		tasks = append(tasks, &TestTask[int]{i, &wgCount, &wgLock, false})
		expectedResult = append(expectedResult, i)
	}
	resultChannel := make(chan *Result[*TaskResult[int]], 1)
	pool := NewFixedPool[int](parallelism, parallelism*2)
	defer pool.Stop()
	go func() {
		results, err := pool.Submit(tasks)
		r.Nil(err)
		resultChannel <- &Result[*TaskResult[int]]{err, results.Results}
	}()
	expired := waitWithTimeout(&wgCount, timeoutDuration)
	r.False(expired, "timed out to wait for tasks equal to parallelism value to be in processing state")
	inProcessStateNumber := 0
	for _, task := range tasks {
		tsk := task.(*TestTask[int])
		if tsk.processed {
			inProcessStateNumber++
		} else {
			tsk.wgCount = nil
			tsk.wgLock = nil
		}
	}
	r.Equal(parallelism, inProcessStateNumber)
	wgLock.Done()
	select {
	case res := <-resultChannel:
		err := res.err
		r.Nil(err)
		results := res.results
		r.NotNil(results)
		var intResults []int
		for _, ts := range results {
			intResults = append(intResults, ts.Result)
		}
		sort.Ints(intResults)
		r.Equal(expectedResult, intResults)
	case <-time.After(timeoutDuration):
		r.Fail("timed out to wait for all tasks to be processed")
	}
}

func TestTestResults(t *testing.T) {
	r := require.New(t)
	var results []*TaskResult[string]
	results = append(results, &TaskResult[string]{
		Result: "test-1",
		Err:    nil,
	})
	err1 := fmt.Errorf("test error 1")
	results = append(results, &TaskResult[string]{
		Result: "test-2",
		Err:    err1,
	})
	taskResults := newTaskResults(results)
	r.NotNil(taskResults)
	stringResults := taskResults.GetResults()
	r.Equal([]string{"test-1", "test-2"}, stringResults)
	errResults := taskResults.GetErrors()
	r.Equal([]error{err1}, errResults)
	r.True(taskResults.HasErrors())
	asSingleError := taskResults.GetAsError()
	r.NotNil(asSingleError)
	r.Equal("1 task(s) failed, errors: \ntest error 1", asSingleError.Error())
}

func TestPoolStop(t *testing.T) {
	r := require.New(t)
	pool := NewFixedPool[string](4, 8)
	var tasks []Task[string]
	for i := 0; i < 4; i++ {
		tasks = append(tasks, &TestTask[string]{"task" + strconv.Itoa(i), nil, nil, false})
	}
	result, err := pool.Submit(tasks)
	r.Nil(err)
	r.Equal(len(tasks), len(result.Results))
	pool.Stop()
	result, err = pool.Submit(tasks)
	r.NotNil(err)
	r.Nil(result)
	r.Equal("pool is stopped", err.Error())
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func TestPool_IsRunning(t *testing.T) {
	r := require.New(t)

	pool := NewFixedPool[string](2, 4)
	defer pool.Stop()

	fixedPool := pool.(*fixedSizePool[string])

	r.True(fixedPool.IsRunning())

	pool.Stop()
	r.False(fixedPool.IsRunning())
}
