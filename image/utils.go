package image

import (
	"net/url"
	"strings"
)

func ScrubIdentifier(identifier string) (string, err) {

	clean, err := url.QueryUnescape(identifier)

	if err != nil {
		return "", err
	}

	clean = strings.Replace(clean, "../", "", -1)
	return clean, nil
}
