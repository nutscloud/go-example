package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
)

type sampleHandler struct{}

func (h *sampleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatal(err)
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}
			//fmt.Printf("Part %q: %q\n", p.Header, slurp)
			fmt.Printf("Part %q: %q\n", p.FileName(), slurp)
		}
	}
}

func main() {
	http.Handle("/", &sampleHandler{})
	fmt.Println(http.ListenAndServe(":8001", nil))
}
