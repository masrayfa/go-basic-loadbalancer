package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type simpleServer struct {
	Addr  string
	Proxy *httputil.ReverseProxy
}

type Server interface {
	Address() string
	IsAlive() bool
	Serve(writer http.ResponseWriter, request *http.Request)
}

func(s *simpleServer) Address() string {
	return s.Addr
}

func(s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(writer http.ResponseWriter, request *http.Request) {
	s.Proxy.ServeHTTP(writer, request)
}

func newSimpleServer(addr string) *simpleServer {
	url, err := url.Parse(addr)
	panicIfError(err)

	return &simpleServer{
		Addr:  addr,
		Proxy: httputil.NewSingleHostReverseProxy(url),
	}
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

type LoadBalancer struct {
	Port string
	RoundRobinCount int
	servers []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		Port: port,
		servers: servers,
		RoundRobinCount: 0,
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.RoundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.RoundRobinCount++
		server = lb.servers[lb.RoundRobinCount%len(lb.servers)]
	}
	lb.RoundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(writer http.ResponseWriter, request *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(writer, request)
	targetServer.Address()
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("https://www.bing.com"),
	}

	lb := NewLoadBalancer("8080", servers)
	handleRedirect := func(writer http.ResponseWriter, request *http.Request) {
		lb.serveProxy(writer, request)
	}

	http.HandleFunc("/", handleRedirect)

	fmt.Println("listening at localhost", lb.Port)
	http.ListenAndServe(":" + lb.Port, nil)
}