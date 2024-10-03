package main

import (
	"sync"

	"github.com/andreysidor4uk/http-gateway-1c/internal/receiver"
	"github.com/andreysidor4uk/http-gateway-1c/internal/responsers/httpresponser"
)

func main() {
	httpResponser := httpresponser.New(":8091")
	receiver := receiver.New(":8090", httpResponser)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		httpResponser.ListenAndServe()
	}()
	go func() {
		defer wg.Done()
		receiver.ListenAndServe()
	}()

	wg.Wait()
}
