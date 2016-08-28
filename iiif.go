package iiif

import (
       "encoding/json"
       "io/ioutil"
)

type Config struct {
     Cache  CacheConfig
}

type CacheConfig struct {
     Name string
     Path string
}

type Cache interface {
     Get(string) ([]byte, error)
     Set(string, []byte) error
     Unset(string) error
}

type Profile interface {

}

func NewConfigFromFile(file string) (*Config, error) {

     body, err := ioutil.ReadFile(file)

     if err != nil {
     	return nil, err
	}

	c := Config{}
	err = json.Unmarshal(body, &c)

	if err != nil {
	   return nil, err
	   }

	   return &c, nil
}