package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfoAsJson(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	url := ts.URL + "/test.png/info.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handle should return JSON by default: got %v want application/json", contentType)
	}
}

func TestInfoAsJsonLd(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/test.png/info.json", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accept", "application/ld+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/ld+json" {
		t.Errorf("handle should return JSON by default: got %v want application/ld+json", contentType)
	}
}

func TestOnlineImage(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	imageURL := "http://dosimple.ch/yoan.png"
	key := base64.StdEncoding.EncodeToString([]byte(imageURL))

	url := ts.URL + "/" + key + "/info.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var m IiifImage
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	if m.Width != 300 && m.Height != 300 {
		t.Errorf("%v image expected to be 300x300: got %v x %v", imageURL, m.Width, m.Height)
	}
}
