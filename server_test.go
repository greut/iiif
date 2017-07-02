package main

import (
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
		t.Errorf("handler should accept bytes ranges: got %v want %v", acceptRanges, "bytes")
	}
}
