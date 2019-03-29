# README

跟著 [Go Phishing！30 天用 Go 實作 Reverse Proxy 之釣魚大作戰](https://ithelp.ithome.com.tw/users/20107714/ironman/1769) 玩一遍

### start

```go
// development
go run main.go

// production
go run main.go --port=:80 --phishURL=https://phish-github.com
```

### admin

```go
// development
http://localhost:8080/phish-admin

// production
https://phish-github.com/phish-admin
```

```go
username: username
password: password
```


