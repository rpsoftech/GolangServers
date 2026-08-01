package main

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

func main() {
	fmt.Println(runtime.GOOS)
	fmt.Println(runtime.GOARCH)
	method := "GET"
	endTo := 184412
	page := 570
	firstValue := 178771
	count := 0
	connectionString := []string{}
	for {
		if firstValue > endTo {
			break
		}
		fmt.Println(firstValue)
		firstValue++
		count++
		connectionString = append(connectionString, fmt.Sprintf("%d", firstValue))
		if count < 10 {
			continue
		}
		client := &http.Client{}
		url := fmt.Sprintf("https://app.wacm.in/contacts/contacts/assigntogroup/%s?group_id=269", strings.Join(connectionString, ","))
		req, err := http.NewRequest(method, url, nil)
		connectionString = []string{}
		page--
		count = 0
		if err != nil {
			fmt.Println(err)
			return
		}
		req.Header.Add("accept", "*/*")
		req.Header.Add("accept-language", "en-IN,en-GB;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Add("priority", "u=1, i")
		req.Header.Add("referer", fmt.Sprintf("https://app.wacm.in/contacts/contacts?page=%d", page))
		req.Header.Add("sec-ch-ua", "\"Google Chrome\";v=\"149\", \"Chromium\";v=\"149\", \"Not)A;Brand\";v=\"24\"")
		req.Header.Add("sec-ch-ua-mobile", "?0")
		req.Header.Add("sec-ch-ua-platform", "\"macOS\"")
		req.Header.Add("sec-fetch-dest", "empty")
		req.Header.Add("sec-fetch-mode", "cors")
		req.Header.Add("sec-fetch-site", "same-origin")
		req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
		req.Header.Add("x-requested-with", "XMLHttpRequest")
		req.Header.Add("Cookie", "remember_web_59ba36addc2b2f9401580f014c7f58ea4e30989d=eyJpdiI6ImFjVGNHWWpjTTBDcjFyd2h1QlluMUE9PSIsInZhbHVlIjoibXB5VW9PMWdnVForbVF5TFY2Nk81QXluVUEzcktUMVg5My96RWdBb2tEUEJwUzEvR2VSV1QzbDFMSGJJRllBT3krK1RveGNLYU1sakJEZ3k3aFZJRzNlcWZOU05mVVg4aHhBOTZwajAydzk2WTE4ekJrZDBINTJTZnJxTHJUYnkydW5aQ0RNTlQ4d2kzSXBBTWhmZjRNSVJ5cjc2VHU3Z2hqYlJHcVlSUXROTmR6bThWeTFSeEZVOGdzTk1KR25GY083V0MybXVaRVJabW9lbGZmdG9TMDh1aHRIWjNlMHlUSkltelBvNXpDMD0iLCJtYWMiOiIzODcxZWEwYTk0M2MxNGFmMGI5OTM1NTBiMjBkMGIwMzA3OTVjOGJjZWUxNjJjMWQyYjg2MzFhYWM4NmM1OTg4IiwidGFnIjoiIn0%3D; lang=eyJpdiI6ImRlZWw2Q0RMeFdhQkI3QVZIRE5Ub1E9PSIsInZhbHVlIjoiSkdVM3VoNkhhRldBMEJ2YUtHMTJkbDUxVW5Mc2FRcHp3emVMWG1XeWF4M3JmS3l1ZWdQc0pzVW1pUlBsY2gzQSIsIm1hYyI6ImJhZjU4OGU3ODQ4OTczZTZmZmRlYmE4NDhhNDhmMTg0NGRlM2FmZDYxZmZkMDE5ZDYyNDAyYmRmZTlkYzUwMjYiLCJ0YWciOiIifQ%3D%3D; XSRF-TOKEN=eyJpdiI6IkpPTCs1dG04UHZtY0ZDaGFTSFpvR1E9PSIsInZhbHVlIjoiZ0VoL2pwQSt0eGRwOWF5UGhLN2pHek9MVXBnZVRIY0JQQ05iUUtWRVlwcTA2LzFpTDlqamR1dFNoK2l6S3JwYWhoaFg0ayszMWUzeVZyM0V3OHVycXdDOW5XdFdrMDNDNTlGMXV0YWRFK1k5cWlhbFpsMmE1MHRrUlZ4bVBWZksiLCJtYWMiOiI5YzBkYjM4MTBkOWFiM2JmNTJiNTM5ZjJkYzM5ODFlOTcwNDgwMmYwODFhOGMwYjg4ZTE5MjM1MDc3MzRhZWUwIiwidGFnIjoiIn0%3D; wacm_whatsapp_campaign_manager_session=eyJpdiI6IlFoMVc1d282dVJjRVJTNHNhWnFLZUE9PSIsInZhbHVlIjoiSW5jY0h6U3ZYRlJzOUVMeFdmWkdsV1ZuN0lLdHp5TXdTdVdmS0xKRUMzL3VxZ2I3cGtMODJEOUJlb3V5ZFduL0R0REp6ZTNsbjMvaU94WC9jU2huZ2RMcFlxT2RsZ0d4RXZKRTA4Q2RERDliNFNWR1M0QXBIV25kVkc1eGZOaFMiLCJtYWMiOiJkOGI3YThmYjdjMzY0ZDZhN2VjNTUzODlmZTEyMjI4MjMzNmIxZmZmODQyODE4NThiMzFkZDUzNDM2Y2I5YWQ3IiwidGFnIjoiIn0%3D")

		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(body))
		time.Sleep(3 * time.Second)
	}
}
