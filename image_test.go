package main

import (
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
	}

	if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "bytes" {
		t.Errorf("handler should accept bytes ranges: got %v want bytes", acceptRanges)
	}
}

func TestRange0To1023(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/test.png/full/full/0/default.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Range", "bytes 0-1023")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusPartialContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusPartialContent)
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "1024" {
		t.Errorf("Content-Length doesn't match: got %v want 1024", contentLength)
	}

	if contentRange := resp.Header.Get("Content-Range"); contentRange != "bytes 0-1023/26427" {
		t.Errorf("Content-Range doesn't match: got %v want bytes 0-1023/26427", contentRange)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if len(body) != 1024 {
		t.Errorf("Received range doesn't match the expected Content-Length: got %v want 1024", len(body))
	}
}

func TestRangeLast1024(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/test.png/full/full/0/default.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Range", "bytes -1024")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusPartialContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusPartialContent)
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "1024" {
		t.Errorf("Content-Length doesn't match: got %v want 1024", contentLength)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if len(body) != 1024 {
		t.Errorf("Received range doesn't match the expected Content-Length: got %v want 1024", len(body))
	}
}

func TestRangeNotSatisfiable(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/test.png/full/full/0/default.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Range", "bytes -30000")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusRequestedRangeNotSatisfiable)
	}

	if contentRange := resp.Header.Get("Content-Range"); contentRange != "bytes */26427" {
		t.Errorf("Content-Range doesn't match: got %v want bytes */26427", contentRange)
	}
}

func TestRangeMultipartNotImplemented(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/test.png/full/full/0/default.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Range", "bytes 0-0,-1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotImplemented)
	}
}
