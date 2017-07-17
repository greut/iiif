package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithGroupCache(t *testing.T) {
	r := makeRouter()
	r = setGroupCache(r, "http://localhost/")
	r = WithRootDirectory(r, "fixtures")
	ts := httptest.NewServer(r)
	defer ts.Close()

	var tests = []struct {
		identifier string
		status     int
	}{
		{"lena.jpg", http.StatusOK},
		{"test.txt", http.StatusNotImplemented},
		{"http://dosimple.ch/yoan.png", http.StatusOK},
		{"http://dosimple.ch", http.StatusNotImplemented},
		{"http://dosimple.ch/missing.png", http.StatusNotFound},
	}

	for _, test := range tests {
		debug("%v ~> %d", test.identifier, test.status)
		url := ts.URL + "/" + test.identifier + "/full/max/0/default.png"
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if status := resp.StatusCode; status != test.status {
			t.Errorf("handler returned wrong status code: got %#v want %#v", status, test.status)
			return
		}
	}
}
