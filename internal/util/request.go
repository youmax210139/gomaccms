package util

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

/*
网络请求, 数据爬取
*/

var (
	Client = CreateClient()
)

// RequestInfo 请求参数结构体
type RequestInfo struct {
	Uri    string      `json:"uri"`    // 请求url地址
	Params url.Values  `json:"param"`  // 请求参数
	Header http.Header `json:"header"` // 请求头数据
	Resp   []byte      `json:"resp"`   // 响应结果数据
	Err    string      `json:"err"`    // 错误信息
}

// CopyRequestInfo 属性复制, 隔离地址引用造成的并发问题
func CopyRequestInfo(r RequestInfo) RequestInfo {
	// 初始化返回值
	newInfo := RequestInfo{Uri: r.Uri, Params: url.Values{}}

	// 循环拷贝r的每个k,v
	for k, v := range r.Params {
		newInfo.Params[k] = append([]string(nil), v...)
	}
	return newInfo
}

// RefererUrl 记录上次请求的url
var RefererUrl string

var respData []byte

// CreateClient 初始化请求客户端
func CreateClient() *colly.Collector {
	c := colly.NewCollector()
	// 访问深度
	c.MaxDepth = 1
	//可重复访问
	c.AllowURLRevisit = true
	// 设置超时时间 默认10s
	c.SetRequestTimeout(20 * time.Second)
	// 发起请求之前会调用的方法
	c.OnRequest(func(request *colly.Request) {
		// 设置一些请求头信息
		request.Headers.Set("Content-Type", "application/json;charset=UTF-8")
		// 请求完成后设置请求头Referer
		if len(RefererUrl) <= 0 || !strings.Contains(RefererUrl, request.URL.Host) {
			RefererUrl = ""
		}
		request.Headers.Set("Referer", RefererUrl)
	})
	// 请求响应成功的回调
	c.OnResponse(func(response *colly.Response) {
		if (response.StatusCode == 200 || (response.StatusCode >= 300 && response.StatusCode <= 399)) && len(response.Body) > 0 {
			// 复制响应内容 (colly 会复用 Body 的底层数组)
			respData = make([]byte, len(response.Body))
			copy(respData, response.Body)
		} else {
			respData = []byte{}
		}
		// 将请求url保存到RefererUrl 用于 Header Refer属性
		RefererUrl = response.Request.URL.String()
	})
	// 请求期间报错的回调
	c.OnError(func(response *colly.Response, err error) {
		log.Printf("请求异常: URL: %s Error: %s\n", response.Request.URL, err)
	})
	return c
}

// ApiGet 请求数据的方法
func ApiGet(r *RequestInfo) {
	// 设置随机请求头
	extensions.RandomUserAgent(Client)
	err := Client.Visit(fmt.Sprintf("%s?%s", r.Uri, r.Params.Encode()))
	if err != nil {
		r.Err = err.Error()
		log.Println("获取数据失败: ", err)
	}
	// 如果没有异常信息则获取请求结果
	r.Resp = respData
}

// ApiTest 处理API请求后的数据, 主测试
func ApiTest(r *RequestInfo) error {
	// 请求成功后的响应
	Client.OnResponse(func(response *colly.Response) {
		// 判断请求状态
		if (response.StatusCode == 200 || (response.StatusCode >= 300 && response.StatusCode <= 399)) && len(response.Body) > 0 {
			// 将响应结构封装到 RequestInfo.Resp中
			r.Resp = response.Body
		} else {
			r.Resp = []byte{}
		}
	})
	// 执行请求返回错误结果
	err := Client.Visit(fmt.Sprintf("%s?%s", r.Uri, r.Params.Encode()))
	log.Println(err)
	return err
}
