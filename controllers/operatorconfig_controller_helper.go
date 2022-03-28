package controllers

import (
	"io/ioutil"
	"strings"
)

func getResources(path string) ([]string, error) {
	var data []byte

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), "---"), nil
}
