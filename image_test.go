package main

import (
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

func TestAcceptRanges(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	url := ts.URL + "/lena.jpg/full/max/0/default.png"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %#v want %#v", status, http.StatusOK)
		return
	}

	if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "bytes" {
		t.Errorf("handler should accept bytes ranges: got %#v want \"bytes\"", acceptRanges)
		return
	}
}

func TestContentDisposition(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	var tests = []struct {
		url    string
		header string
	}{
		{"/lena.jpg/full/max/0/default.png", "inline; filename=lena.jpg-full-max-0-default.png"},
		{"/lena.jpg/full/max/0/default.png?dl", "attachement; filename=lena.jpg-full-max-0-default.png"},
	}
	for _, test := range tests {
		url := ts.URL + test.url
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != test.header {
			t.Errorf("Content-Disposition should enable downloading, got: %#v want %#v", contentDisposition, test.header)
			return
		}
	}
}

func TestOutputSizes(t *testing.T) {
	ts := newServer()
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
		{"/lena.jpg/full/!400,300/0/default.png", 140, 300},
		{"/lena.jpg/full/pct:50/0/default.png", 542, 1159},
		{"/lena.jpg/square/max/0/default.png", 1084, 1084},
		{"/lena.jpg/square/500,500/0/default.png", 500, 500},
		{"/lena.jpg/square/500,/0/default.png", 500, 500},
		{"/lena.jpg/square/,500/0/default.png", 500, 500},
		{"/lena.jpg/smart/500,500/0/default.png", 500, 500},
		{"/lena.jpg/84,318,1000,2000/max/0/default.png", 1000, 2000},
		{"/lena.jpg/84,318,1000,2000/500,1000/0/default.png", 500, 1000},
		{"/lena.jpg/84,318,1000,2000/500,/0/default.png", 500, 1000},
		{"/lena.jpg/84,318,1000,2000/,1000/0/default.png", 500, 1000},
		{"/lena.jpg/pct:10,10,80,80/max/0/default.png", 867, 1854},
		{"/lena.jpg/0,0,1084,2318/256,/0/default.png", 256, 547},
		{"/lena.jpg/0,0,1084,2318/512,/0/default.png", 512, 1094},
		{"/lena.jpg/542,1159,542,1159/512,/0/default.png", 512, 1094},
		{"/lena.jpg/84,313,1000,2000/pct:50/0/default.png", 500, 1000},
	}

	for _, test := range tests {
		debug("%s ~> %d x %d", test.url, test.width, test.height)
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
			t.Errorf("handler returned wrong status code: got %v want %v\nmessage: %s", status, http.StatusOK, string(body))
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

func TestFailing(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	var tests = []struct {
		url    string
		status int
	}{
		{"/lena.jpg/full/max/0/default.png", 200},
		{"/lena.jpg/full/max/1/default.png", 501},
		{"/lena.jpg/full/max/0/bitonal.png", 501},
		{"/lena.jpg/full/pct:-1/0/default.png", 400},
		{"/lena.jpg/full/10/0/default.png", 400},
		{"/lena.jpg/full/10,10,10/0/default.png", 400},
		{"/lena.jpg/10/max/0/default.png", 400},
		{"/lena.jpg/10,10/max/0/default.png", 400},
		{"/lena.jpg/10,10,10/max/0/default.png", 400},
		{"/lena.jpg/10,10,10,10,10/max/0/default.png", 400},
		{"/lena.jpg/-10,10,10,10/max/0/default.png", 400},
		{"/lena.jpg/10,10,0,0/max/0/default.png", 400},
		{"/lena.jp2/full/max/0/default.png", 404},
	}

	for _, test := range tests {
		debug("%s ~> %d", test.url, test.status)
		url := ts.URL + test.url
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		if status := resp.StatusCode; status != test.status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.status)
			return
		}
	}
}
