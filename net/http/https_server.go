package main

import (
	"fmt"
	"net/http"
)

// $ openssl genrsa -out server.key 2048
// $ openssl req -new -key server.key > server.csr
// $ openssl req -x509 -days 365 -key server.key -in server.csr > server.crt
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w,
		"Hi, This is an example of https service in golang!")
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServeTLS(":8081", "server.crt",
		"server.key", nil)
}
