package main

import (
	"fmt"
	"log"
	"net/url"
	"flag"

	//"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	//"local/gunlib/topicline"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var cookies []*http.Cookie

var login string
var pass string

type topciline struct {
	name  string
	href  string
	id    string
	owner string
}

func loginFn() []*http.Cookie {
	resp1, _ := http.PostForm("https://forum.guns.ru/forum/login", url.Values{"UserName": {login}, "Password": {pass}})
	defer resp1.Body.Close()
	cookies = resp1.Cookies()

	if len(cookies) == 0 {
		panic("empty cookies")
	}

	fmt.Println(cookies)

	return cookies
}

func getDoc(auth []*http.Cookie) *goquery.Document {
	var ret *goquery.Document = nil

	req, err := http.NewRequest("GET", "https://forum.guns.ru/forumtopics/25.html", nil)
	if err != nil {
			return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
			return nil
	}

	defer resp.Body.Close()

	utf8, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		fmt.Println("Encoding error:", err)
		return nil
	}

	ret, err = goquery.NewDocumentFromReader(utf8)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	
	nodes := ret.Find("a[name='open_login']")

	fmt.Println("Auth: ", nodes.Size())

	if nodes.Size() != 0 {
		time.Sleep(time.Second)
		ret = getDoc(loginFn())
	} 
	return ret
}

func getnewtopics(lastid string) ([]topciline, error) {
	topics := []topciline{}


	doc := getDoc(cookies)

	isStop := false

	doc.Find("table.topicline").Each(func(i int, selection *goquery.Selection) {
		topic := topciline{}

		isNeedAdd := false
		isNeedAdd2 := true

		selection.Find("td[width='50%']").Each(func(i int, tmpsel *goquery.Selection) {
			tmpsel.Find("a").Each(func(i int, s *goquery.Selection) {
				if i == 0 {
					name := s.Text()
					href, _ := s.Attr("href")
					indexS := strings.LastIndex(href, "/")
					indexE := strings.LastIndex(href, "-")
					if indexE == -1 {
						indexE = strings.LastIndex(href, ".")
					}
					id := href[indexS+1 : indexE]

					topic.name = name
					topic.href = href
					topic.id = id
				}
			})

		})

		selection.Find("td[width='15%']").Each(func(i int, tmpsel *goquery.Selection) {
			if tmpsel.Text() == " " {
				isNeedAdd = true
			}
		})

		selection.Find("td[width='20%']").Each(func(i int, tmpsel *goquery.Selection) {
			if strings.Contains(tmpsel.Text(), "важно") {
				isNeedAdd2 = false
			}
		})

		topic.owner = selection.Find("td[width='12%']").Find("nobr").Text()

		if topic.id == lastid {
			isStop = true
		}

		if isNeedAdd && isNeedAdd2 && !isStop {
			topics = append(topics, topic)
		}
	})

	return topics, nil
}

func main() {

	flag.StringVar(&login, "login", "", "login")
	flag.StringVar(&pass, "pass", "", "password")
	tokenPtr := flag.String("token", "", "token")
	chatId := flag.Int64("id", 0, "chatID")

	//237031995

	cookies = loginFn()

	ret, _ := getnewtopics("")

	lastid := ret[0].id
	fmt.Println(lastid)
	fmt.Println(ret[0].href)
	fmt.Println(len(ret))

	bot, err := tgbotapi.NewBotAPI(*tokenPtr)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	
	ticker := time.NewTicker(60 * time.Second)

	for _ = range ticker.C {
		rett, _ := getnewtopics(lastid)
		//fmt.Println(rett[0].id)
		fmt.Println(len(rett))

		if len(rett) != 0 {
			for i, topic := range rett {
				if i == 0 {
					lastid = topic.id
				}
				msg := tgbotapi.NewMessage(*chatId, topic.name + "\n" + topic.href)
				bot.Send(msg)
			}
		}

	}

}
