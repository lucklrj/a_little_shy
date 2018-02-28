package main

import (
	"bufio"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/parnurzeal/gorequest"
	"os"
	"strings"
	"regexp"
	"strconv"
	"flag"
)

type shyData struct {
	Title             string
	WebLink           string
	AuthorPublishDate string
	Authorame         string
	AuthorLink        string
	AuthorAvatarLink  string
	Content           string
}

func (s shyData) Do() {
	fmt.Println(s.Title)
}

var (
	maxPage        = 1
	startPage      = 1
	Request        *gorequest.SuperAgent
	captchaCode    string
	captchaID      string
	DoubanAccount  string
	DoubanPassword string
	inputAccount   = flag.String("a", "", "登陆豆瓣的账号")
	inputPassword  = flag.String("p", "", "登陆豆瓣的密码")
)

func init() {
	Request = gorequest.New()
	flag.Parse()
	DoubanAccount = *inputAccount
	DoubanPassword = *inputPassword
	
}
func main() {
	login()
	for {
		color.Green("正在获取第" + strconv.Itoa(startPage) + "页数据。")
		listUrl := makeListUrl(startPage)
		ids, errs := getViewIds(listUrl)
		if errs != nil {
			outputAllErros(errs, false)
			
		} else {
			idsLength := len(ids)
			list := make([]chan shyData, idsLength)
			for i := 0; i < idsLength; i++ {
				list[i] = make(chan shyData)
				go getContent(ids[i], list[i])
			}
			
			for _, ch := range list {
				singleData := <-ch
				singleData.Do()
			}
		}
		startPage = startPage + 1
		if startPage > maxPage {
			color.Red("已达到最大页数限制")
			os.Exit(0)
		}
	}
}
func checkAccountandPassword() {
	if DoubanAccount == "" {
		color.Red("请输入豆瓣的登陆账户")
		fmt.Scanln(&DoubanAccount)
	}
	if DoubanPassword == "" {
		color.Red("请输入豆瓣的登陆密码")
		fmt.Scanln(&DoubanPassword)
	}
}
func login() {
	checkAccountandPassword()
	
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
		
		color.Red("请输入图片验证码：")
		fmt.Scanln(&captchaCode)
	}
	color.Green("开始登陆")
	
	postData := make(map[string]string)
	postData["form_email"] = DoubanAccount
	postData["form_password"] = DoubanPassword
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
		//检查失败原因 验证码不正确
		if strings.Contains(html, "验证码不正确") == true {
			color.Red("验证码不正确,请重试")
		}
		if strings.Contains(html, "帐号和密码不匹配") == true {
			DoubanPassword = ""
			color.Red("帐号和密码不匹配,请重试")
		}
		if strings.Contains(html, "该用户不存在") == true {
			DoubanAccount = ""
			color.Red("帐号不存在,请重试")
		}
		login()
	}
}

func makeListUrl(page int) string {
	url := "https://www.douban.com/group/haixiuzu/discussion?start=" + strconv.Itoa((page-1)*25)
	return url
}
func getViewIds(listUrl string) (ids []string, err []error) {
	ids = make([]string, 0)
	_, body, errs := Request.Get(listUrl).End()
	if errs != nil {
		return nil, errs
	} else {
		//开始提取[ids]
		doc, err := goquery.NewDocumentFromReader(bufio.NewReader(strings.NewReader(body)))
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		doc.Find(".title").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the band and title
			url, _ := s.Find("a").Attr("href")
			reg, _ := regexp.Compile(`.*\/([0-9]+)\/`)
			match := reg.FindAllStringSubmatch(url, -1)
			if len(match) == 1 {
				ids = append(ids, match[0][1])
			}
		})
		return ids, nil
	}
}
func getContent(id string, c chan shyData) {
	
	singleData := shyData{}
	singleData.Title = "aa:" + id
	c <- singleData
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
