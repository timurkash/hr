package main

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ЗАДАНИЕ:
// * сделать из плохого кода хороший;
// * важно сохранить логику появления ошибочных тасков;
// * сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через merge-request.

// приложение эмулирует получение и обработку тасков, пытается и получать и обрабатывать в многопоточном режиме
// В конце должно выводить успешные таски и ошибки выполнения остальных тасков

// A TType represents a meaninglessness of our life
type TType struct {
	id         int
	createTime *time.Time // Время создания
	finishTime *time.Time // Время выполнения
	err        error
	taskResult []byte
}

const (
	SucceededConst     = "succeeded"
	TaskSucceededConst = "task has been " + SucceededConst
	WentWrongConst     = "something went wrong"
)

var (
	Succeeded         = []byte(SucceededConst)
	TaskSucceeded     = []byte(TaskSucceededConst)
	WentWrong         = []byte(WentWrongConst)
	SomeErrorOccurred = errors.New("some error occurred")
)

var (
	taskCreateFunc = func(ch chan<- *TType) {
		go func(ch chan<- *TType) {
			i := 0
			for {
				ft := time.Now()
				i++
				tType := &TType{
					//id:         int(time.Now().Unix()),
					id:         i,
					createTime: &ft,
				} // передаем таск на выполнение
				if time.Now().Nanosecond()%2 > 0 { // вот такое условие появления ошибочных тасков
					tType.err = SomeErrorOccurred
				}
				ch <- tType
			}
		}(ch)
	}
	taskWorkerFunc = func(tType *TType) *TType {
		if tType.createTime.After(time.Now().Add(-20 * time.Second)) {
			tType.taskResult = TaskSucceeded
		} else {
			tType.taskResult = WentWrong
		}
		now := time.Now()
		tType.finishTime = &now
		time.Sleep(time.Millisecond * 150)
		return tType
	}
	taskSorterFunc = func(tType *TType, doneTasks chan<- *TType, undoneTasks chan<- error) {
		if bytes.HasSuffix(tType.taskResult, Succeeded) {
			doneTasks <- tType
		} else {
			undoneTasks <- fmt.Errorf("task id %d time %s, error %s",
				tType.id, tType.createTime, tType.taskResult)
		}
	}
)

var (
	superChan   = make(chan *TType, 10)
	doneTasks   = make(chan *TType)
	undoneTasks = make(chan error)
	results     = make(map[int]*TType)
	errs        []error
	m           sync.Mutex
)

func main() {
	go taskCreateFunc(superChan)
	go func(superChan chan *TType, doneTasks chan<- *TType, undoneTasks chan<- error) {
		// получение тасков
		for task := range superChan {
			taskSorterFunc(taskWorkerFunc(task), doneTasks, undoneTasks)
		}
		close(superChan)
	}(superChan, doneTasks, undoneTasks)
	go func() {
		for doneTask := range doneTasks {
			go func(tType *TType) {
				m.Lock()
				results[doneTask.id] = doneTask
				m.Unlock()
			}(doneTask)
		}
		for undoneTask := range undoneTasks {
			go func(err error) {
				errs = append(errs, undoneTask)
			}(undoneTask)
		}
		close(doneTasks)
		close(undoneTasks)
	}()
	time.Sleep(time.Second * 3)
	printResults(results, errs)
}

func printResults(results map[int]*TType, errs []error) {
	println("Done tasks:")
	for i, result := range results {
		fmt.Printf("%d %d %s\n", i, result.id, result.taskResult)
	}
	println("Errors:")
	for err := range errs {
		println(err)
	}
}
