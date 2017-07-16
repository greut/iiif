package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetHtml(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	var tests = []struct {
		url string
	}{
		{"/"},
		{"/demo"},
		{"/lena.jpg/iiifviewer.html"},
		{"/lena.jpg/leaflet.html"},
		{"/lena.jpg/openseadragon.html"},
	}

	for _, test := range tests {
		url := ts.URL + test.url
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if status := resp.StatusCode; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
			t.Errorf("index should return HTML by default: got %v want text/html", contentType)
		}
	}
}

func TestRedirectToInfo(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/images/test.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("X-Forwarded-Host", "example.org")
	req.Header.Add("X-Forwarded-Proto", "https")

	client := &http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if status := resp.StatusCode; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}

	if location := resp.Header.Get("Location"); location != "https://example.org/images/test.png/info.json" {
		t.Errorf("Location returned bad value: got %#v", location)
	}
}

func TestInfoAsJson(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	url := ts.URL + "/images/test.png/info.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handle should return JSON by default: got %v want application/json", contentType)
	}
}

func TestInfoImageID(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/images/test.png/info.json", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("X-Forwarded-Host", "example.org")
	req.Header.Add("X-Forwarded-Proto", "https")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var m IiifImage
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasPrefix(m.ID, "https://example.org") {
		t.Errorf("Image ID expected to contains correct host name, got: %v", m.ID)
	}
}

func TestInfoAsJsonLd(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/images/test.png/info.json", nil)
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

func TestOnlineImageBase64(t *testing.T) {
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

func TestOnlineImageUrl(t *testing.T) {
	r := makeRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	var tests = []struct {
		url    string
		width  int
		height int
	}{
		{"http://dosimple.ch/yoan.png", 300, 300},
		{"http://loremflickr.com/320/240?random=1", 320, 240},
	}

	for _, test := range tests {
		resp, err := http.Get(ts.URL + "/" + url.QueryEscape(test.url) + "/info.json")
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

		if m.Width != test.width && m.Height != test.height {
			t.Errorf("%v image expected to be %dx%d: got %dx%d", test.url, test.width, test.height, m.Width, m.Height)
		}
	}
}
