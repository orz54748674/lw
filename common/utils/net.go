package utils

import (
	"io/ioutil"
	"net/http"
)

func Request(client *http.Client, res *http.Request) (body []byte, err error) {
	res.Header.Add("Accept", "application/json")
	resp, err := client.Do(res)
	if err != nil {
		return
	}
	return ioutil.ReadAll(resp.Body)
}
