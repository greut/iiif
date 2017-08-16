package iiif

import (
	"encoding/base64"
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetHtml(t *testing.T) {
	ts := newServer()
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
	ts := newServer()
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

func TestEtag(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	url := ts.URL + "/images/test.png/full/max/0/default.png"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if etag := resp.Header.Get("ETag"); etag == "" {
		t.Errorf("handle should have a ETag header, got nothing.")
	}
}

func TestInfoAsJson(t *testing.T) {
	ts := newServer()
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

func TestInfo(t *testing.T) {
	ts := newServer()
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

	var m Image
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasPrefix(m.ID, "https://example.org") {
		t.Errorf("Image ID expected to contains correct host name, got: %v", m.ID)
	}

	var p ImageProfile
	_ = mapstructure.Decode(m.Profile[1], &p)

	if p.MaxArea != 0 {
		t.Errorf("Profile MaxArea expected to be missing, got: %v.", p.MaxArea)
	}
}

func TestInfoMaxSize(t *testing.T) {
	ts := newServerWithMaxSize(400, 200, 50000)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/images/test.png/info.json", nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var m Image
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	var p ImageProfile
	_ = mapstructure.Decode(m.Profile[1], &p)

	if p.MaxWidth != 400 {
		t.Errorf("Profile MaxArea expected to be 400, got: %v.", p.MaxWidth)
	}
	if p.MaxHeight != 200 {
		t.Errorf("Profile MaxArea expected to be 200, got: %v.", p.MaxHeight)
	}
	if p.MaxArea != 50000 {
		t.Errorf("Profile MaxArea expected to be 50000, got: %v.", p.MaxArea)
	}
}

func TestInfoAsJsonLd(t *testing.T) {
	ts := newServer()
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
	ts := newServer()
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

	var m Image
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	if m.Width != 300 && m.Height != 300 {
		t.Errorf("%v image expected to be 300x300: got %v x %v", imageURL, m.Width, m.Height)
	}
}

func TestOnlineImageUrl(t *testing.T) {
	ts := newServer()
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

		var m Image
		err = decoder.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}

		if m.Width != test.width && m.Height != test.height {
			t.Errorf("%v image expected to be %dx%d: got %dx%d", test.url, test.width, test.height, m.Width, m.Height)
		}
	}
}

func newServer() *httptest.Server {
	return newServerWithMaxSize(0, 0, 0)
}

func newServerWithMaxSize(width, height, area int) *httptest.Server {
	r := MakeRouter()
	r = WithConfig(r, &Config{
		Images:    "../fixtures",
		Templates: "../templates",
		MaxArea:   area,
		MaxWidth:  width,
		MaxHeight: height,
	})
	return httptest.NewServer(r)
}
