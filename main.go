package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var wg sync.WaitGroup

// Request is used for making requests to services behind a load balancer.
type Request struct {
	Payload interface{}
	RspChan chan Response
}

// Response is the value returned by services behind a load balancer.
type Response interface{}

// LoadBalancer is used for balancing load between multiple instances of a service.
type LoadBalancer interface {
	Request(payload interface{}) chan Response
	RegisterInstance(chan Request)
}

// MyLoadBalancer is the load balancer you should modify!
type MyLoadBalancer struct {
	mutex            *sync.Mutex
	requestsChannels []*RequestChannel
	current          uint64
}

type RequestChannel struct {
	ch chan Request
}

// NextRequestChannel atomically increase the counter and return an next RequestChannel.
// using Round Robin Selection.
// Round Robin is simple. It gives equal opportunities for workers to perform tasks in turns.
func (lb *MyLoadBalancer) NextRequestChannel() (*RequestChannel, error) {
	currentIndex := int(lb.current)
	for i := currentIndex; i < currentIndex+len(lb.requestsChannels); i++ {
		slicedIndex := i % len(lb.requestsChannels)
		requestsChannel := lb.requestsChannels[slicedIndex]
		atomic.AddUint64(&lb.current, uint64(1))
		return requestsChannel, nil
	}
	return nil, fmt.Errorf("no request a live")
}

// Request is currently a dummy implementation. Please implement it!
func (lb *MyLoadBalancer) Request(payload interface{}) chan Response {
	ch := make(chan Response, 1)

	if len(lb.requestsChannels) == 0 { // in case that no time service instance
		fmt.Println("no time service instance")
		ts := manager.Spawn()
		lb.RegisterInstance(ts.ReqChan)
		wg.Add(1)
		go ts.Run()
	}

	lb.mutex.Lock()
	requestChannel, err := lb.NextRequestChannel()
	lb.mutex.Unlock()

	if err != nil {
		ch <- err.Error()
		return ch
	}
  fmt.Printf("requestChannel : %#v selected \n",requestChannel)
	respChan := make(chan Response, 1)
	requestChannel.ch <- Request{Payload: payload, RspChan: respChan}
	for {
		select {
		case res := <-respChan:
			ch <- res
			return ch
		}
	}
}

// RegisterInstance is currently a dummy implementation. Please implement it!
func (lb *MyLoadBalancer) RegisterInstance(ch chan Request) {

	lb.mutex.Lock() // to make that load balancer is accessible
	defer lb.mutex.Unlock()
	fmt.Printf("RegisterInstance : %#v \n", ch)
	lb.requestsChannels = append(lb.requestsChannels, &RequestChannel{
		ch: ch,
	})
	return
}

/******************************************************************************
 *  STANDARD TIME SERVICE IMPLEMENTATION -- MODIFY IF YOU LIKE                *
 ******************************************************************************/

// TimeService is a single instance of a time service.
type TimeService struct {
	Dead            chan struct{}
	ReqChan         chan Request
	AvgResponseTime float64
}

// Run will make the TimeService start listening to the two channels Dead and ReqChan.
func (ts *TimeService) Run() {
	for {
		select {
		case <-ts.Dead:
			return
		case req := <-ts.ReqChan:
			processingTime := time.Duration(ts.AvgResponseTime+1.0-rand.Float64()) * time.Second
			time.Sleep(processingTime)
			req.RspChan <- time.Now()
		}
	}
}

/******************************************************************************
 *  CLI -- YOU SHOULD NOT NEED TO MODIFY ANYTHING BELOW                       *
 ******************************************************************************/

var manager *TimeServiceManager // to make accessible for all package

// main runs an interactive console for spawning, killing and asking for the
// time.
func main() {
	rand.Seed(int64(time.Now().Nanosecond()))

	bio := bufio.NewReader(os.Stdin)

	manager = &TimeServiceManager{}

	var lb LoadBalancer = &MyLoadBalancer{
		mutex:            &sync.Mutex{},
		requestsChannels: []*RequestChannel{},
	}

	for {
		fmt.Printf("> ")
		cmd, err := bio.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading command: ", err)
			continue
		}
		switch strings.TrimSpace(cmd) {
		case "kill":
			manager.Kill()
		case "spawn":
			ts := manager.Spawn()
			lb.RegisterInstance(ts.ReqChan)
			wg.Add(1)
			go ts.Run()
		case "time":
			select {
			case rsp := <-lb.Request(nil):
				fmt.Println(rsp)
			case <-time.After(5 * time.Second):
				fmt.Println("Timeout")
			}
		default:
			fmt.Printf("Unknown command: %s Available cocmdmmands: time, spawn, kill\n", cmd)
		}
	}

	wg.Wait() // 4
	fmt.Println("main finished")
}

// TimeServiceManager is responsible for spawning and killing.
type TimeServiceManager struct {
	Instances []TimeService
}

// Kill makes a random TimeService instance unresponsive.
func (m *TimeServiceManager) Kill() {
	if len(m.Instances) > 0 {
		n := rand.Intn(len(m.Instances))
		close(m.Instances[n].Dead)
		m.Instances = append(m.Instances[:n], m.Instances[n+1:]...)
	}
}

// Spawn creates a new TimeService instance.
func (m *TimeServiceManager) Spawn() TimeService {
	ts := TimeService{
		Dead:            make(chan struct{}, 0),
		ReqChan:         make(chan Request, 10),
		AvgResponseTime: rand.Float64() * 3,
	}
	m.Instances = append(m.Instances, ts)

	return ts
}
