package main

import (
	"io/ioutil"
	"net/http"
	v8 "github.com/ry/v8worker"
	"fmt"
	"runtime"
	"time"
	"sync"
	"log"
	"strconv"
)
var jsCode []byte;
var WorkerQueue chan WorkerPoolMember
var tasksQueue chan WorkRequest
var RequestQueue int
var mutex sync.Mutex

type WorkRequest struct {
	w http.ResponseWriter
	r *http.Request
	out chan JavascriptResponse
}

type WorkerPoolMember struct {
	worker *v8.Worker
	workerNumber int
	chi <-chan JavascriptResponse
	busy bool
}


type JavascriptResponse struct {
	Type string
	Payload string
	ContentType string
}

func (worker WorkerPoolMember) DoInWorker(out chan JavascriptResponse){
	go func(){
		//log.Printf("Load worker number: %s", strconv.Itoa(worker.workerNumber))
		//start := time.Now()
		code := string(jsCode)
		compileError := worker.worker.Load("code.js", code)
		//elapsed := time.Since(start)
		if compileError != nil{
			log.Printf("Compile error: %s", compileError)
		}
		return

	}()
	for {
		runtime.Gosched()
		select {
		case jsRsp := <- worker.chi:
			out <- jsRsp
			return
		default:
			time.Sleep(1 * time.Microsecond);
			continue
		}
	}
}

func listenChannels () {
	tasksQueue = make(chan WorkRequest, 1000)
	for {
		runtime.Gosched()
		select {
		case workRequest := <-tasksQueue:
			runtime.Gosched()
			go func (){
				for {
					select {
					case worker := <-WorkerQueue:
						workerFound(worker,workRequest)
						return
					default:
						time.Sleep(1 * time.Microsecond);
						continue
					}
				}

			}()

		default:
			time.Sleep(1 * time.Microsecond);
			continue
		}
	}
}


func handleRequest(w http.ResponseWriter, req *http.Request){
	out := make(chan JavascriptResponse)
	workRequest := WorkRequest{w,req,out}
	tasksQueue <- workRequest
	for {
		runtime.Gosched()
		select {
		case jsRsp := <-out:
			w.Header().Set("Content-Type", jsRsp.ContentType)
			w.Header().Set("Connection", "close")
			w.Write([]byte(jsRsp.Payload))
			//hijack,_ := w.(http.Hijacker)
			//conn, _, _ := hijack.Hijack();
			//defer conn.Close();
			//bufrw.Flush();
			close(out)
			return
		default:
			time.Sleep(1 * time.Microsecond);
			continue
		}
	}
}

func workerFound(worker WorkerPoolMember, workRequest WorkRequest){
	out := make(chan JavascriptResponse)
	runtime.Gosched()
	go worker.DoInWorker(out)
	for {
		runtime.Gosched()
		select {
		case jsRsp := <-out:
			close(out)
			workRequest.out <- jsRsp
			worker.worker.TerminateExecution()
			WorkerQueue <- worker
			return
		default:
			time.Sleep(1 * time.Microsecond);
			continue
		}
	}
}

func NewWorker(workerNumber int) WorkerPoolMember {
	out := make(chan JavascriptResponse)

	//var worker *v8.Worker
	worker, _ := v8.New(func(msg string) {
		jsRsp := JavascriptResponse{"exit", msg, "text/plain"}
		out <- jsRsp
		return
	},func(msg string) string {
		return "krai"
	})

	// Create, and return the worker.
	workerMember :=  WorkerPoolMember{
		worker,
		workerNumber,
		out,
		false,
	}

	return workerMember
}

func StartDispatcher(nworkers int) {
	// First, initialize the channel we are going to but the workers' work channels into.
	WorkerQueue = make(chan WorkerPoolMember, nworkers)

	// Now, create all of our workers.
	for i := 0; i<nworkers; i++ {

		worker := NewWorker(i)
		WorkerQueue <- worker
	}
	fmt.Println("Workers Started", nworkers)
	return
}

func MaxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func main() {
	log.Print(strconv.Itoa(MaxParallelism()))
	MaxParallelismProcesses := MaxParallelism()
	runtime.GOMAXPROCS(MaxParallelismProcesses)
	var err error
	jsCode, err = ioutil.ReadFile("./code.js")
	if err != nil {
		panic(err);
	}
	StartDispatcher(MaxParallelismProcesses*20)
	go listenChannels()
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRequest)
	http.ListenAndServe(":8080", mux)

}