```go
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

```

1. NextRequestChannel atomically increase the counter and return an next RequestChannel.
2. using Round Robin Selection (It gives equal opportunities for workers to perform tasks in turns).
3. [Round Robin Selection link](https://morioh.com/p/db6cac742e1c).

```go
func (lb *MyLoadBalancer) Request(payload interface{}) chan Response {
	ch := make(chan Response, 1)

	if len(lb.requestsChannels) == 0 { // in case that no time service instance
    //create new  TimeService is a single instance to ensure that user get response.
		fmt.Println("no time service instance")
		ts := manager.Spawn()
		lb.RegisterInstance(ts.ReqChan)
		wg.Add(1)
		go ts.Run()
	}

	lb.mutex.Lock()//to make sure that only one processs select the  NextRequestChannel
	requestChannel, err := lb.NextRequestChannel()
	lb.mutex.Unlock()

	if err != nil {
		ch <- err.Error()
		return ch
	}

    fmt.Printf("requestChannel : %#v selected \n",requestChannel)

    //
	respChan := make(chan Response, 1)
    //create Request instance 
	requestChannel.ch <- Request{Payload: payload, RspChan: respChan}
	for {
		select {
		case res := <-respChan:
			ch <- res
			return ch
		}
	}
}
```

The get a payload Select the new request channel and creat response channels to get the result.

```go
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
```

add new instance of req channel into load balancer.