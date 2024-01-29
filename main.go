package main

import (
	"fmt"
	"strings"
	"github.com/gocolly/colly/v2"
	"sync"
	"strconv"
)

type Job struct {
	Title			string
	Employer		string
	Link			string
}

func main() {
	fmt.Println("main start")
	keywords := []string{"software", "engineer"}
	beginScrapping(keywords)
	fmt.Println("main end")
}

func beginScrapping(keywords []string) {
	fmt.Println("begin scrapping")
	ch := make(chan Job)
	var wg sync.WaitGroup
	wg.Add(1)
	go ScrapMeroJob(keywords, ch, &wg)

	go func(){
		wg.Wait()
		close(ch)
	}()

	for jobs := range ch {
		fmt.Printf("Job details: title %v employer %v link %v \n", jobs.Title, jobs.Employer, jobs.Link)
	}

	fmt.Println("end scrappin")
}

// func ScrapJobNepal(keywords []string,wg *sync.WaitGroup) {
// 	defer (*wg).Done()
// 	fmt.Println("scrap start")
// 	url := fmt.Sprintf(`https://www.jobsnepal.com/search?q=%s`, strings.Join(keywords, "+"))
// 	c := colly.NewCollector()

// 	c.OnHTML("div.vb-content", func(e *colly.HTMLElement) {
// 		fmt.Println("job title is ", e.Text)
// 	})

// 	c.OnRequest(func(r *colly.Request) {
// 		fmt.Println("Visiting", r.URL)
// 	})

// 	c.Visit(url)
// 	c.Wait()
// }

func ScrapMeroJob(keywords []string, ch chan Job, wg *sync.WaitGroup) {
	domain := "https://merojob.com"
	defer (*wg).Done()
	fmt.Println("scrap mero job start")
	url := fmt.Sprintf(`%s/search/?q=%s`, domain, strings.Join(keywords, "+"))
	c := colly.NewCollector()

	c.OnHTML("#search_job", func(e *colly.HTMLElement) {
		e.ForEach("div.card", func(_ int, el *colly.HTMLElement){
			id := el.Attr("id")
			// some card have id that aren't job
			if id == "" {
			goquerySelection := el.DOM
			jobTitleElement := goquerySelection.Find("h1")
			jobLinkElement := jobTitleElement.Find("a")
			jobTitle := jobLinkElement.Text()
			employerName := goquerySelection.Find("h3 > a").Text()
			jobLink, _ := jobLinkElement.Attr("href")

			job := Job{
				Title: strings.TrimSpace(jobTitle),
				Employer: strings.TrimSpace(employerName),
				Link: jobLink,
			}

			ch <- job
		}
		})
	})

	c.OnHTML("a.page-link", func (e *colly.HTMLElement) {
		linkText := e.Text
		linkN, err := strconv.Atoi(linkText)
		if err == nil && linkN > 1 {
			link := domain + e.Attr("href")
			c.Visit(link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit(url)
	c.Wait()
}