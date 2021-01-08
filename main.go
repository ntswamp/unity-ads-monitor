package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const ORGANIZATIONID = "15668055009592"
const APIKEY = "bb7d0bfc404e46906b4f37be3fa9822c977bf752658ab650c3d17e62ab433fbf"

type AdState struct {
	AdRequest int `json:"adrequest_count"`
}

func main() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		bBlocked, ad, err := isBlocked()
		if err != nil {
			//place holder
			log.Fatal(err)
		}
		if bBlocked && ad == 0 {
			done <- true
			defer close(done)
		}
	}()
	for {
		select {
		case <-done:
			fmt.Println("広告が停止しています")
			//send to chatwork
			return
		case <-ticker.C:
			fmt.Println("異常ありません")
		}
	}
}

func isBlocked() (bool, int, error) {
	//time setup
	now := time.Now().UTC()
	h, _, _ := now.Clock()
	rewind2hour := ToBaseTime(now, h-2, 0, 0)
	start := rewind2hour.Format(time.RFC3339)
	end := now.Format(time.RFC3339)
	log.Println("start time:", start, "\tend time:", end)

	//set up request
	url := "https://monetization.api.unity.com/stats/v1/operate/organizations/" + ORGANIZATIONID
	param := map[string]string{
		"fields": "adrequest_count",
		"scale":  "all",
		"start":  start,
		"end":    end,
		"apikey": APIKEY,
	}
	header := map[string]string{
		"Accept": "application/json",
	}

	//if get() failed retry 5 times at a 10 seconds interval
	var httperr error = nil
	var resp *http.Response
	for i := 0; i < 6; i++ {
		resp, httperr = get(url, param, header)
		if httperr == nil {
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				error.Error(err)
			}
			body = bytes.Trim(body, "[]")
			var msg AdState
			json.Unmarshal(body, &msg)
			log.Println("adrequest_count:", msg.AdRequest)
			//we are blocked
			if msg.AdRequest == 0 {
				return true, msg.AdRequest, nil
			}
			//no exception
			return false, msg.AdRequest, nil
		}
		//retry 5 times
		if i != 5 {
			log.Println("networking error, retry in 10 seconds.")
			time.Sleep(time.Second * 10)
			continue
		} else {
			log.Println("network error. restart program manually.")
			break
		}
	}
	//network error
	return false, -1, httperr
}

func get(url string, param map[string]string, header map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("new request is fail ")
	}
	//add params
	q := req.URL.Query()
	if param != nil {
		for key, val := range param {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
	//add headers
	if header != nil {
		for key, val := range header {
			req.Header.Add(key, val)
		}
	}
	//http client
	client := &http.Client{}
	//log.Printf("Go %s URL : %s \n", http.MethodGet, req.URL.String())
	return client.Do(req)
}

// ToBaseTime ...
func ToBaseTime(now time.Time, hour, min, sec int) time.Time {
	target := now.Add(-(time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second))
	dt := time.Date(target.Year(), target.Month(), target.Day(), hour, min, sec, 0, target.Location())
	return dt
}
