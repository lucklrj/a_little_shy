package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/parnurzeal/gorequest"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/gjson"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type shyData struct {
	DouBanId          string
	FetchResult       string
	Title             string
	AuthorPublishDate string
	AuthorName        string
	AuthorLink        string
	AuthorAvatarLink  string
	Content           string
}

func (s shyData) Do() {
	//fmt.Println("DouBanId" + ":" + s.DouBanId)
	//fmt.Println("FetchResult" + ":" + s.FetchResult)
	//fmt.Println("Title" + ":" + s.Title)
	//fmt.Println("AuthorPublishDate" + ":" + s.AuthorPublishDate)
	//fmt.Println("AuthorName" + ":" + s.AuthorName)
	//fmt.Println("AuthorLink" + ":" + s.AuthorLink)
	//fmt.Println("AuthorAvatarLink" + ":" + s.AuthorAvatarLink)
	//fmt.Println("Content" + ":" + s.Content)
	//fmt.Println("-------------------------------------")
	if s.FetchResult == "success" {
		AllShyData = append(AllShyData, s)
	}
}

type HttpRequest struct {
	Request *gorequest.SuperAgent
	mu      sync.Mutex
}

func (h *HttpRequest) Get(url string) (body string, errs []error) {
	h.mu.Lock()
	_, body, errs = h.Request.Get(url).End()
	h.mu.Unlock()
	return body, errs
}
func (h *HttpRequest) Post(url string, postData map[string]string) (body string, errs []error) {
	h.mu.Lock()
	_, body, errs = h.Request.Post(url).Type("multipart").Send(postData).End()
	h.mu.Unlock()
	return body, errs
}

var (
	startPage           = 1
	captchaCode         string
	captchaID           string
	DoubanAccount       string
	DoubanPassword      string
	DataReceiveUrl      string
	MaxPage             int
	inputAccount        = flag.String("douban-account", "", "登陆豆瓣的账号")
	inputPassword       = flag.String("douban-password", "", "登陆豆瓣的密码")
	inputDataReceiveUrl = flag.String("remote-url", "", "采集数据接收的远端地址")
	inputMaxPage        = flag.String("max-page", "1", "单次采集最大页数")
	DB                  *leveldb.DB
	Request             HttpRequest
	AllShyData          []shyData
)

func init() {
	Request = HttpRequest{Request: gorequest.New()}
	flag.Parse()
	DoubanAccount = *inputAccount
	DoubanPassword = *inputPassword
	DataReceiveUrl = *inputDataReceiveUrl
	MaxPage, _ = strconv.Atoi(*inputMaxPage)
	
	DB = initDB()
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
			ids = filterIds(DB, ids)
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
		if startPage > MaxPage {
			color.Green("已达到最大页数限制")
			postToRemote()
			break
		}
	}
	defer DB.Close()
	os.Exit(0)
}
func initDB() *leveldb.DB {
	DB, err := leveldb.OpenFile("db", nil)
	if err != nil {
		color.Red(err.Error())
		os.Exit(0)
	}
	return DB
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
	if DataReceiveUrl == "" {
		color.Red("请输入数据接收地址")
		fmt.Scanln(&DataReceiveUrl)
	}
}
func login() {
	checkAccountandPassword()
	
	color.Green("检查是否需要输入验证码")
	html, errs := Request.Get("https://accounts.douban.com/login")
	if errs != nil {
		outputAllErros(errs, true)
	}
	if html == "Please try later." {
		color.Red("请求频繁，被豆瓣限制.")
		os.Exit(0)
	}
	
	isNeedCheck := false
	if strings.Contains(html, "captcha-id") == true {
		isNeedCheck = true
		r, _ := regexp.Compile(`<input.*?name="captcha\-id".*?value="(.*?)"\/>`)
		match := r.FindAllStringSubmatch(html, -1)
		captchaID = match[0][1]
		
		imgContent, errs := Request.Get("https://www.douban.com/misc/captcha?id=" + captchaID + "&size=s")
		if errs != nil {
			outputAllErros(errs, true)
		}
		saveFile("captcha_id.jpg", imgContent, false)
		
		color.Red("请输入图片验证码：")
		fmt.Scanln(&captchaCode)
	}
	color.Green("开始登陆")
	
	postData := make(map[string]string)
	postData["ck"] = ""
	postData["name"] = DoubanAccount
	postData["password"] = DoubanPassword
	postData["redir"] = "http://www.douban.com/group/"
	postData["ticket"] =""
	postData["remember"] = "on"
	if isNeedCheck == true {
		postData["captcha-id"] = captchaID
		postData["captcha-solution"] = captchaCode
	}
	html, errs = Request.Post("https://accounts.douban.com/j/mobile/login/basic", postData)
	if errs != nil {
		outputAllErros(errs, true)
	}

	description :=gjson.Parse(html).Get("description").String()


	if description == "处理成功"{
		color.Green("登陆成功")
	} else {
		//检查失败原因 验证码不正确
		color.Red(description)

		//login()
	}
}
func makeListUrl(page int) string {
	url := "https://www.douban.com/group/haixiuzu/discussion?start=" + strconv.Itoa((page-1)*25)
	return url
}
func getViewIds(listUrl string) (ids []string, err []error) {
	ids = make([]string, 0)
	body, errs := Request.Get(listUrl)
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
func filterIds(db *leveldb.DB, ids []string) []string {
	filterIds := make([]string, 0)
	for _, singleID := range (ids) {
		isExists, _ := db.Has([]byte(singleID), nil)
		if isExists == true {
			continue
		} else {
			filterIds = append(filterIds, singleID)
		}
	}
	return filterIds
}
func getContent(id string, c chan shyData) {
	singleData := shyData{}
	singleData.DouBanId = id
	
	html, errs := Request.Get("https://www.douban.com/group/topic/" + id + "/")
	if errs != nil {
		outputAllErros(errs, false)
		singleData.FetchResult = "获取html内容错误"
		c <- singleData
		return
	}
	
	doc, err := goquery.NewDocumentFromReader(bufio.NewReader(strings.NewReader(html)))
	if err != nil {
		color.Red(err.Error())
		singleData.FetchResult = "解析html文档错误"
		c <- singleData
		return
	}
	
	//获取主题
	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		singleData.Title = trim(s.Text())
	})
	
	// AuthorPublishDate 作者发布时间
	doc.Find(".color-green").Each(func(i int, s *goquery.Selection) {
		singleData.AuthorPublishDate = trim(s.Text())
	})
	//AuthorName, 作者名.AuthorLink 作者连接
	doc.Find(".from a").Each(func(i int, s *goquery.Selection) {
		AuthorName := s.Text()
		singleData.AuthorName = trim(AuthorName)
		
		AuthorLink, _ := s.Attr("href")
		singleData.AuthorLink = AuthorLink
	})
	//AuthorAvatarLink, 作者头像连接
	doc.Find(".topic-content .user-face .pil").Each(func(i int, s *goquery.Selection) {
		AuthorAvatarLink, _ := s.Attr("src")
		singleData.AuthorAvatarLink = AuthorAvatarLink
	})
	
	//Content 获取内容
	doc.Find(".topic-richtext").Each(func(i int, s *goquery.Selection) {
		Content, _ := s.Html()
		singleData.Content = trim(clearTags(Content))
	})
	if singleData.Content == "" {
		doc.Find(".topic-figure").Each(func(i int, s *goquery.Selection) {
			Content, _ := s.Html()
			singleData.Content = trim(clearTags(Content))
		})
	}
	
	if strings.Contains(singleData.Content, "<img") == true {
		singleData.FetchResult = "success"
	} else {
		singleData.FetchResult = "noImage"
	}
	
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
func trim(s string) string {
	s = strings.Trim(s, "\n")
	s = strings.Trim(s, " ")
	return s
}
func clearTags(str string) string {
	str = strings.Replace(str, "\n", "", -1)
	str = regexp.MustCompile(`<\/?div.*?>`).ReplaceAllString(str, "")
	str = regexp.MustCompile(`<\/?p.*?>`).ReplaceAllString(str, "")
	str = regexp.MustCompile(`<img.*?src=([\"\'])(.*?)([\"\']).*?\/?>`).ReplaceAllString(str, "<img src=$1$2$3 />")
	return str
}
func postToRemote() {
	if len(AllShyData) > 0 {
		color.Green("正在上传数据")
		_, body, errs := Request.Request.Post(DataReceiveUrl).Type("multipart").Send(AllShyData).End()
		if errs != nil {
			outputAllErros(errs, true)
		}
		json := gjson.Parse(body)
		if json.Get("code").Int() == 200 {
			
			color.Green("同步成功:", json.Get("run_num").Int())
			for _, singleShyData := range AllShyData {
				if singleShyData.FetchResult == "success" || singleShyData.FetchResult == "noImage" {
					err := DB.Put([]byte(singleShyData.DouBanId), []byte("1"), nil)
					if err != nil {
						fmt.Println(err.Error())
					}
				}
			}
		} else {
			color.Red("同步失败:" + body)
			
		}
	} else {
		color.Green("没有新数据")
	}
}
