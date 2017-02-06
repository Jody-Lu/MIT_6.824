package mapreduce

import (
	"fmt"
	"sync"
	//"time"
)

//
// schedule() starts and waits for all tasks in the given phase (Map
// or Reduce). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//

func putAddrBack(addr string, ch chan string) {
	ch <- addr
}

func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var n_other int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mapFiles)
		n_other = nReduce
	case reducePhase:
		ntasks = nReduce
		n_other = len(mapFiles)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO
	// Two worker threads

	file_idx := 0
	task_idx := 0
	numMapFile := len(mapFiles)
	numWorker := 2

	for file_idx < numMapFile && task_idx < ntasks {
		var wg sync.WaitGroup
		var addrs []string

		for i := 0; i < numWorker && file_idx < numMapFile && task_idx < ntasks; i++ {
			addr := <-registerChan
			dotask := DoTaskArgs{jobName, mapFiles[file_idx], phase, task_idx, n_other}
			ch := make(chan bool)
			wg.Add(1)

			go func(addr string, dotask DoTaskArgs, ch chan bool) {
				ch <- call(addr, "Worker.DoTask", dotask, nil)
				wg.Done()
			}(addr, dotask, ch)

			file_idx++
			task_idx++

			if !(<-ch) { // <-ch == false: failure
				file_idx--
				task_idx--
			}

			addrs = append(addrs, addr)

			if file_idx >= numMapFile || task_idx >= ntasks {
				return
			}
		}

		wg.Wait()

		for i := 0; i < numWorker; i++ {
			go putAddrBack(addrs[i], registerChan)
		}

	}
	fmt.Printf("Schedule: %v phase done\n", phase)
}
