package main

import (
	"io"
	"net/http"
	"strconv"
	"sync"
)

type ReqContainer struct {
	req  *http.Request
	w    http.ResponseWriter
	done chan struct{}
}

type ResQueue struct {
	m      sync.Mutex
	lastId uint64
	q      map[string]*ReqContainer
}

func (rq *ResQueue) Push(reqC *ReqContainer) string {
	rq.m.Lock()
	defer rq.m.Unlock()

	rq.lastId++
	id := strconv.FormatUint(rq.lastId, 36)

	rq.q[id] = reqC

	return id
}

func (rq *ResQueue) Pop(id string) *ReqContainer {
	rq.m.Lock()
	defer rq.m.Unlock()

	reqC := rq.q[id]
	delete(rq.q, id)

	return reqC
}

var reqQueue = make(chan *ReqContainer)
var resQueue = ResQueue{
	q: make(map[string]*ReqContainer),
}

func anyRequest(w http.ResponseWriter, req *http.Request) {
	reqC := ReqContainer{
		req:  req,
		w:    w,
		done: make(chan struct{}),
	}

	select {
	case <-req.Context().Done():
	case reqQueue <- &reqC:
	}

	select {
	case <-req.Context().Done():
	case <-reqC.done:
	}
}

func request(w http.ResponseWriter, req *http.Request) {
	select {
	case <-req.Context().Done():
	case reqC := <-reqQueue:
		for key, headers := range reqC.req.Header {
			for _, value := range headers {
				w.Header().Add(key, value)
			}
		}

		id := resQueue.Push(reqC)
		w.Header().Add("x-req-id", string(id))
		w.Header().Add("x-req-url", reqC.req.RequestURI)

		defer reqC.req.Body.Close()
		body, _ := io.ReadAll(reqC.req.Body)
		w.Write(body)
	}
}

func response(w http.ResponseWriter, req *http.Request) {
	id := req.Header.Get("x-req-id")
	reqC := resQueue.Pop(id)

	select {
	case <-req.Context().Done():
	default:
		for key, headers := range req.Header {
			for _, value := range headers {
				reqC.w.Header().Add(key, value)
			}
		}

		defer req.Body.Close()
		io.Copy(w, req.Body)
		reqC.done <- struct{}{}
	}
}

func main() {
	http.HandleFunc("/", anyRequest)
	http.HandleFunc("/request", request)
	http.HandleFunc("/response", response)
	http.ListenAndServe(":8090", nil)
}
