package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithGroupCache(t *testing.T) {
	r := makeHandler("http://localhost/")
	ts := httptest.NewServer(r)
	defer ts.Close()

	var tests = []struct {
		identifier string
	}{
		{"lena.jpg"},
		{"http://dosimple.ch/yoan.png"},
	}

	for _, test := range tests {
		url := ts.URL + "/" + test.identifier + "/full/max/0/default.png"
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if status := resp.StatusCode; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %#v want %#v", status, http.StatusOK)
			return
		}
	}
}
