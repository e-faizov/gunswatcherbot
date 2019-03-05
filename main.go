package main

import (
	"fmt"
	"log"
	"net/url"
	"flag"
	"errors"
	"strconv"

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

var m map[string]bool

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

func getDoc(auth []*http.Cookie) (*goquery.Document, error) {
	var ret *goquery.Document = nil

	req, err := http.NewRequest("GET", "https://forum.guns.ru/forumtopics/25.html", nil)
	if err != nil {
			return nil, err
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
			return nil, err
	}

	defer resp.Body.Close()

	utf8, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		fmt.Println("Encoding error:", err)
		return nil, err
	}

	ret, err = goquery.NewDocumentFromReader(utf8)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	
	nodes := ret.Find("a[name='open_login']")

	fmt.Println("Auth: ", nodes.Size())

	if nodes.Size() != 0 {
		return nil, errors.New("Auth fail")
	} 
	return ret, nil
}

func getnewtopics() ([]topciline, error) {
	topics := []topciline{}


	doc, err := getDoc(cookies)
	if err != nil {
		return nil, err
	}

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

		if isNeedAdd && isNeedAdd2 && !isStop {
			_, ok := m[topic.id]
			if !ok {
				topics = append(topics, topic)
				m[topic.id] = true
			}

		}
	})

	return topics, nil
}

func main() {

	m = make(map[string]bool)
	flag.StringVar(&login, "login", "", "login")
	flag.StringVar(&pass, "pass", "", "password")
	tokenPtr := flag.String("token", "", "token")
	chatId := flag.Int64("id", 0, "chatID")

	flag.Parse()

	if login == "" || pass == "" || *tokenPtr == "" || *chatId == 0 {
		log.Panic()
	}

	cookies = loginFn()

	_, err := getnewtopics()

	bot, err := tgbotapi.NewBotAPI(*tokenPtr)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	
	ticker := time.NewTicker(60 * time.Second)

	for _ = range ticker.C {
		rett, err := getnewtopics()

		if err != nil {
			msg := tgbotapi.NewMessage(*chatId, "error: " + err.Error())
			bot.Send(msg)
		} else if len(rett) != 0 {
			sDelim := "----------------------------------"

			sMsg := "count: " + strconv.Itoa(len(rett)) + "\n" + sDelim
			for _, topic := range rett {
				sMsg += topic.name + "\n" + topic.href + "\n" + sDelim
			}

			msg := tgbotapi.NewMessage(*chatId, sMsg)
			bot.Send(msg)
		}

	}
}
