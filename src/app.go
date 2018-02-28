package main

import (
	"github.com/parnurzeal/gorequest"
	"github.com/fatih/color"
	"strconv"
	"os"
	"strings"
	"regexp"
	"fmt"
)

var (
	maxPage     = 1
	startPage   = 1
	Request     *gorequest.SuperAgent
	captchaCode string
	captchaID   string
)

func init() {
	Request = gorequest.New()
}
func main() {
	login()
	
	for {
		color.Green("正在获取第" + strconv.Itoa(startPage) + "页数据。")
		listUrl := makeListUrl(startPage)
		_, errs := getViewIds(listUrl)
		if errs != nil {
			outputAllErros(errs, false)
		} else {
		
		}
		startPage = startPage + 1
		if startPage > maxPage {
			color.Red("已达到最大页数限制")
			os.Exit(0)
		}
	}
}

func login() {
	color.Green("检查是否需要输入验证码")
	_, html, errs := Request.Get("http://www.douban.com/").End()
	if errs != nil {
		outputAllErros(errs, true)
	}
	
	isNeedCheck := false
	if strings.Contains(html, "captcha") == true {
		isNeedCheck = true
		r, _ := regexp.Compile(`<input.*?name="captcha\-id".*?value="(.*?)"\/>`)
		match := r.FindAllStringSubmatch(html, -1)
		captchaID = match[0][1]
		
		_, imgContent, errs := Request.Get("http://www.douban.com/misc/captcha?id=" + captchaID + "&size=s").End()
		if errs != nil {
			outputAllErros(errs, true)
		}
		saveFile("captcha_id.jpg", imgContent, false)
		
		color.Green("请输入图片验证数字：")
		fmt.Scanln(&captchaCode)
	}
	color.Green("开始登陆")
	
	postData := make(map[string]string)
	postData["form_email"] = "sunny_lrj@yeah.net"
	postData["form_password"] = "123asd123"
	postData["redir"] = "http://www.douban.com/group/"
	postData["source"] = "group"
	postData["user_login"] = "登录"
	postData["remember"] = "on"
	if isNeedCheck == true {
		postData["captcha-id"] = captchaID
		postData["captcha-solution"] = captchaCode
	}
	
	_, html, errs = Request.Post("https://www.douban.com/accounts/login").Type("multipart").Send(postData).End()
	if errs != nil {
		outputAllErros(errs, true)
	}
	if strings.Contains(html, "我的小组话题") {
		color.Green("登陆成功")
	} else {
		color.Red("登陆失败，请重试")
		login()
	}
}

func makeListUrl(page int) string {
	url := "https://www.douban.com/group/haixiuzu/discussion?start=" + strconv.Itoa((page-1)*25)
	return url
}
func getViewIds(listUrl string) (ids []string, err []error) {
	ids = make([]string, 25)
	_, _, errs := Request.Get(listUrl).End()
	if errs != nil {
		return nil, errs
	} else {
		return ids, nil
	}
}

func outputAllErros(errs []error, end bool) {
	for _, err := range (errs) {
		color.Red(err.Error())
	}
	if end == true {
		os.Exit(0)
	}
}
func saveFile(path string, content string, append bool) (result bool, err error) {
	way := 0
	if append == true {
		way = os.O_RDWR | os.O_CREATE | os.O_APPEND
	} else {
		way = os.O_RDWR | os.O_CREATE
	}
	fd, err := os.OpenFile(path, way, 0644)
	buf := []byte(content)
	fd.Write(buf)
	defer fd.Close()
	return true, err
}
