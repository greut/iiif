package main

import (
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAcceptRanges(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	url := ts.URL + "/test.png/full/full/0/default.png"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		return
	}

	if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "bytes" {
		t.Errorf("handler should accept bytes ranges: got %v want bytes", acceptRanges)
		return
	}
}

func TestOutputSizes(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	var tests = []struct {
		url    string
		width  int
		height int
	}{
		{"/lena.jpg/full/max/0/default.png", 1084, 2318},
		{"/lena.jpg/full/max/90/default.png", 2318, 1084},
		{"/lena.jpg/full/max/!90/default.png", 2318, 1084},
		{"/lena.jpg/full/max/180/default.png", 1084, 2318},
		{"/lena.jpg/full/max/!180/default.png", 1084, 2318},
		{"/lena.jpg/full/max/270/default.png", 2318, 1084},
		{"/lena.jpg/full/max/!270/default.png", 2318, 1084},
		{"/lena.jpg/full/400,300/0/default.png", 400, 300},
		{"/lena.jpg/full/!400,300/0/default.png", 187, 300},
		{"/lena.jpg/full/pct:50/0/default.png", 542, 1159},
		{"/lena.jpg/square/max/0/default.png", 1084, 1084},
		{"/lena.jpg/square/500,500/0/default.png", 500, 500},
		{"/lena.jpg/square/500,/0/default.png", 500, 500},
		{"/lena.jpg/square/,500/0/default.png", 500, 500},
		{"/lena.jpg/smart/500,500/0/default.png", 500, 500},
		{"/lena.jpg/84,318,1000,2000/max/0/default.png", 1000, 2000},
		{"/lena.jpg/84,318,1000,2000/500,1000/0/default.png", 500, 1000},
		{"/lena.jpg/pct:10,10,80,80/max/0/default.png", 867, 1854},
	}

	for _, test := range tests {
		url := ts.URL + test.url
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		if status := resp.StatusCode; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v\nmesage: %s", status, http.StatusOK, string(body))
			return
		}

		image := bimg.NewImage(body)
		size, err := image.Size()
		if err != nil {
			log.Fatal(err)
		}

		if size.Width != test.width || size.Height != test.height {
			t.Errorf("sizes do not match for %v: got %vx%v want %vx%v", test.url, size.Width, size.Height, test.width, test.height)
			return
		}
	}
}
