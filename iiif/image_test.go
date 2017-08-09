package iiif

import (
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
		{"/lena.jpg/full/max/0/default.jpeg", 1084, 2318},
		{"/lena.jpg/full/max/0/default.jpg", 1084, 2318},
		{"/lena.jpg/full/max/0/default.webp", 1084, 2318},
		{"/lena.jpg/full/max/0/default.tiff", 1084, 2318},
		{"/lena.jpg/full/max/0/default.tif", 1084, 2318},
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

func TestOutputMaxSizes(t *testing.T) {
	ts := newServerWithMaxSize(200, 300, 50000)
	defer ts.Close()

	var tests = []struct {
		url    string
		width  int
		height int
	}{
		{"/lena.jpg/full/max/0/default.png", 140, 300},
		{"/lena.jpg/square/max/0/default.png", 200, 200},
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
	ts := newServerWithMaxSize(2000, 3000, 5000000)
	defer ts.Close()

	var tests = []struct {
		url    string
		status int
	}{
		{"/lena.jpg/full/max/0/default.png", http.StatusOK},
		{"/lena.jpg/full/max/0/default.gif", http.StatusNotImplemented},
		{"/lena.jpg/full/max/0/default.pdf", http.StatusNotImplemented},
		{"/lena.jpg/full/max/0/default.jp2", http.StatusNotImplemented},
		{"/lena.jpg/full/max/0/default.svg", http.StatusNotImplemented},
		{"/lena.jpg/full/max/0/default.pdf", http.StatusNotImplemented},
		{"/lena.jpg/full/max/0/default.bmp", http.StatusNotImplemented},
		{"/lena.jpg/full/max/1/default.png", http.StatusNotImplemented},
		{"/lena.jpg/full/max/flip/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/max/0/bitonal.png", http.StatusNotImplemented},
		{"/lena.jpg/full/pct:-1/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/10/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/10,10,10/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/10/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/10,10/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/10,10,10/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/10,10,10,10,10/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/-10,10,10,10/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/0,0,10000,10000/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/10,10,0,0/max/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/2001,10/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/10,3001/0/default.png", http.StatusBadRequest},
		{"/lena.jpg/full/2000,3000/0/default.png", http.StatusBadRequest},
		{"/lena.jp2/full/max/0/default.png", http.StatusNotFound},
		{"/lena.jpg/index.html", http.StatusNotFound},
		{"/images/full/max/0/default.png", http.StatusBadRequest},
		{"/images/info.json", http.StatusBadRequest},
		{"/lena.jp2/info.json", http.StatusNotFound},
		{"/test.txt/full/max/0/default.png", http.StatusNotImplemented},
		{"/test.txt/info.json", http.StatusNotImplemented},
		{"/" + url.QueryEscape("http://dosimple.ch/missing.png") + "/full/max/0/default.png", http.StatusNotFound},
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
			t.Errorf("handler returned wrong status code: got %v want %v for %v", status, test.status, test.url)
			return
		}
	}
}
