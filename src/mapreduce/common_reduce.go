package mapreduce

import (
	"bufio"
	"encoding/json"
	//"fmt"
	"os"
	"sort"
)

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// doReduce manages one reduce task: it reads the intermediate
// key/value pairs (produced by the map phase) for this task, sorts the
// intermediate key/value pairs by key, calls the user-defined reduce function
// (reduceF) for each key, and writes the output to disk.
func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTaskNumber int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	//
	// You will need to write this function.
	//
	// You'll need to read one intermediate file from each map task;
	// reduceName(jobName, m, reduceTaskNumber) yields the file
	// name from map task m.
	//
	// Your doMap() encoded the key/value pairs in the intermediate
	// files, so you will need to decode them. If you used JSON, you can
	// read and decode by creating a decoder and repeatedly calling
	// .Decode(&kv) on it until it returns an error.
	//
	// You may find the first example in the golang sort package
	// documentation useful.
	//
	// reduceF() is the application's reduce function. You should
	// call it once per distinct key, with a slice of all the values
	// for that key. reduceF() returns the reduced value for that key.
	//
	// You should write the reduce output as JSON encoded KeyValue
	// objects to the file named outFile. We require you to use JSON
	// because that is what the merger than combines the output
	// from all the reduce tasks expects. There is nothing special about
	// JSON -- it is just the marshalling format we chose to use. Your
	// output code will look something like this:
	//
	// enc := json.NewEncoder(file)
	// for key := ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()

	// nMap: number of map task
	var pairs []KeyValue

	// Read all intermediate data
	for m := 0; m < nMap; m++ {
		file_name := reduceName(jobName, m, reduceTaskNumber)
		file, _ := os.Open(file_name)
		defer file.Close()

		scanner := bufio.NewScanner(file)

		// Decode JSON into []KeyValue
		for scanner.Scan() {
			var s KeyValue
			json.Unmarshal([]byte(scanner.Text()), &s)
			pairs = append(pairs, s)
		}
	}

	// Sort to merge
	sort.Sort(ByKey(pairs))

	// Open output file
	f, _ := os.Create(outFile)

	// Create JSON encoder
	enc := json.NewEncoder(f)

	i := 0
	j := 0

	// Do not use mapping because pairs is sorted
	for j < len(pairs) {
		var values []string
		for j < len(pairs) && pairs[i].Key == pairs[j].Key {
			values = append(values, pairs[j].Value)
			j++
		}
		enc.Encode(KeyValue{pairs[i].Key, reduceF(pairs[i].Key, values)})
		i = j
	}

	f.Close()

}
