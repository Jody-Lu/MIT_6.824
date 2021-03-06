##**6.824 2017 Lecture 2: Infrastructure: RPC and threads**

**Most commonly-asked question: Why Go?**
  
  - 6.824 used to use C++
    students spent time fixing bugs unrelated to distributed systems
      e.g., they freed objects that were still in use
  - Go allows you to concentrate on distributed systems problems
    - type safe
    - garbage-collected (no use after freeing problems)
    - good support for concurrency
    - good support for RPC
  - We like programming in Go
    - simple language to learn
  After the tutorial, use https://golang.org/doc/effective_go.html

**Threads**
  
  - threads are a useful structuring tool
  **Go calls them goroutines**; everyone else calls them threads
  they can be tricky

**Why are threads?**
  
  - Allow you to exploit concurrency, which shows up naturally in distributed systems
  - I/O concurrency:
    While waiting for a response from another server, process next request
  - Multicore:
    While working on one request on one core, process another one on an idle core
  - Both show up in the crawler exercise

**Thread = "thread of execution"**
  
  - threads allow one program to (logically) execute many things at once
  - the threads share memory
  - each thread includes some per-thread state:
    - program counter, registers, stack

**How many threads in a program?**
  
  - As many as "useful" for the application
  - Go encourages you to create many threads
    - Typically more threads than cores
    - The Go runtime schedules them on the available cores
  - Go threads are **not free**, but you should think as if they are
    - **Creating a thread is more expensive** than a method call
    
**Threading challenges:**
  
  - sharing data 
     - two threads modify the same variable at same time?
     - one thread reads data that another thread is changing?
     - these problems are often called **races**
     - need to protect **invariants** on shared data
     - use Go **sync.Mutex**
  - coordination between threads
    - e.g. wait for all Map threads to finish
    - use **Go channels** or **WaitGroup**
  - deadlock 
     - thread 1 is waiting for thread 2
     - thread 2 is waiting for thread 1
     - easy detectable (unlike races)
  - lock granularity
     - coarse-grained -> simple, but little concurrency/parallelism
     - fine-grained -> more concurrency, more races and deadlocks
  
**Crawler exercise**
  
  - Two opportunities for parallelism:
    - While fetching an URL, work on another URL
    - Work on different URLs on different cores
  - Challenge: fetch each URL once
    - want to avoid wasting network bandwidth
    - want to be nice to remote servers
    => Need to some way of keeping track which URLs visited 
    
**Solutions**

  - Sequential one: pass fetched two recursive calls
    - Doesn't overlap I/O when fetcher takes a long time
    - Doesn't exploit multiple cores
  - Solution with a **go routines** and shared fetched
    - Create a thread for each url
      What happens we don't pass u?  (race)
    - Why locks?  (remove them and every appears to works!)
      - What can go wrong without locks?
        Checking and marking of url isn't atomic
	- So it can happen that we fetch the same url twice.
	T1 checks fetched[url], T2 checks fetched[url]
	Both see that url hasn't been fetched
	Both return false and both fetch
      - This called a **race condition**
        - The bug shows up only with some thread interleavings
	Hard to find, hard to reason about
      - Go can detect races for you (go run -race crawler.go)
    - How can we decide we are done with processing a page?
      - waitGroup
  - Solution with channels instead of WaitGroup
    - Channels: general-purse mechanism to coordinate threads
      - bounded-buffer of messages
      - several threads can send and receive on channel
        (Go runtime uses locks internally)
    - Sending or receiving can block
      - when channel is full
      - when channel is empty
  - Solution with only channels
    - Dispatch every url through a filter thread
    - Termination using a separate channel

**What is the best solution?**

  - All concurrent ones are substantial more difficult than serial one
  - Some Go designers argue avoid shared-memory
    - I.e. Use channels only
  - My lab solutions use many concurrency features
    - locks for shared-memory data structures
    - channels for coordination
    - WaitGroup for determining threads are done
  - Use Go's race detector:
    - https://golang.org/doc/articles/race_detector.html
    - go test -race mypkg

**Remote Procedure Call (RPC)**

  - a key piece of distributed system machinery; all the labs use RPC
  - goal: easy-to-program network communication
    hides most details of client/server communication
    client call is much like ordinary procedure call
    server handlers are much like ordinary procedures
  - RPC is widely used!

**RPC ideally makes net communication look just like fn (function) call:**

```  
  Client:
    z = fn(x, y)
  
  Server:
    fn(x, y) {
      compute
      return z
    }
```
  **RPC aims for this level of transparency**

- Go example: kv.go
  - Client "dials" server and invokes Call()
    Call similar to a regular function call
  - Server handles each request in a separate thread
    Concurrency!  Thus, locks for keyvalue.

- RPC message diagram:

```
  Client             Server
    request--->
       <---response
```
- Software structure
  
```
  client app         handlers
    stubs           dispatcher
   RPC lib           RPC lib
     net  ------------ net
```

- A few details:
  - Which server function (handler) to call?
    - In Go specified in Call()
  - Marshalling: format data into packets
    - Tricky for arrays, pointers, objects, &c
    - Go's RPC library is pretty powerful!
    - some things you cannot pass: e.g., **channels, functions**
  - Binding: how does client know who to talk to?
    - Maybe client supplies server host name
    - Maybe a name service maps service names to best server host

**RPC problem: what to do about failures?**
  
  - e.g. lost packet, broken network, slow server, crashed server

- What does a failure look like to the client RPC library?
  - Client never sees a response from the server
  - Client does **not** know if the server saw the request!
    - Maybe server/net failed just before sending reply
  [diagram of lost reply]

- Simplest scheme: "at least once" behavior
  - RPC library waits for response for a while
  - If none arrives, re-send the request
  - Do this a few times
  - Still no response -- return an error to the application

**Q: is "at least once" easy for applications to cope with?**

- Simple problem w/ at least once:
  - client sends "deduct $10 from bank account"

**Q: what can go wrong with this client program?**
  
  ```
  Put("k", 10) -- an RPC to set key's value in a DB server
  Put("k", 20) -- client then does a 2nd Put to same key
  [diagram, timeout, re-send, original arrives very late]
  ```
  
**Q: is at-least-once ever OK?**
  
  - yes: if it's OK to repeat operations, e.g. read-only op
  - yes: if application has its own plan for coping w/ duplicates
    which you will need for Lab 1

**Better RPC behavior: "at most once"**

  - idea: server RPC code detects duplicate requests
    returns previous reply instead of re-running handler
  - **Q: how to detect a duplicate request?**
  client includes unique ID (XID) with each request
    uses same XID for re-send
  server:
  
  ```
    if seen[xid]:
      r = old[xid]
    else
      r = handler()
      old[xid] = r
      seen[xid] = true
  ```
  
**some at-most-once complexities**
  
  - this will come up in labs 3 and on
  - how to ensure XID is unique?
    - big random number?
    - combine unique client ID (ip address?) with sequence #?
  - server must eventually discard info about old RPCs
    - when is discard safe?
    - idea:
      - unique client IDs
      - per-client RPC sequence numbers
      - client includes "seen all replies <= X" with every RPC
      - much like TCP sequence #s and acks
    - or only allow client one outstanding RPC at a time
      - arrival of seq+1 allows server to discard all <= seq
    - or client agrees to keep retrying for < 5 minutes
      - server discards after 5+ minutes
  - how to handle dup req while original is still executing?
    - server doesn't know reply yet; don't want to run twice
    - idea: "pending" flag per executing RPC; wait or ignore

**What if an at-most-once server crashes and re-starts?**

  - if at-most-once duplicate info in memory, server will forget
    and accept duplicate requests after re-start
  maybe it should write the duplicate info to disk?
  maybe replica server should also replicate duplicate info?

**What about "exactly once"?**

  - at-most-once plus unbounded retries plus fault-tolerant service
  Lab 3

**Go RPC is "at-most-once"**
  
  - open TCP connection
  - write request to TCP connection
  - TCP may retransmit, but server's TCP will filter out duplicates
  - no retry in Go code (i.e. will NOT create 2nd TCP connection)
  - Go RPC code returns an error if it doesn't get a reply
    - perhaps after a timeout (from TCP)
    - perhaps server didn't see request
    - perhaps server processed request but server/net failed before reply came back

**Go RPC's at-most-once isn't enough for Lab 1**
  
  - it only applies to a single RPC call
  - if worker doesn't respond, the master re-send to it to another worker
    but original worker may have not failed, and is working on it too
  - Go RPC can't detect this kind of duplicate
    No problem in lab 1, which handles at application level
    Lab 3 will explicitly detect duplicates