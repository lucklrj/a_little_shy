package main

import (
	"github.com/parnurzeal/gorequest"
	"github.com/fatih/color"
	"strconv"
	"os"
)

var (
	maxPage   = 10
	startPage = 1
	Request   *gorequest.SuperAgent
)

func init() {
	Request = gorequest.New()
}
func main() {
	
	for {
		color.Green("正在获取第" + strconv.Itoa(startPage) + "页数据。")
		listUrl := makeListUrl(startPage)
		_, errs := getViewIds(listUrl)
		if errs != nil {
			for _, err := range (errs) {
				color.Red(err.Error())
			}
		} else {
		
		}
		startPage = startPage + 1
		if startPage > maxPage {
			color.Red("已达到最大页数限制")
			os.Exit(0)
		}
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
