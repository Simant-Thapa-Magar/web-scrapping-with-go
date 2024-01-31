package main

import (
	"fmt"
	"strings"
	"github.com/gocolly/colly/v2"
	"sync"
	"strconv"
	"context" 
	"github.com/chromedp/cdproto/cdp" 
	"github.com/chromedp/chromedp" 
	"log" 
	"time"
)

type Job struct {
	Title			string
	Employer		string
	Link			string
	Source			string
}

func main() {
	fmt.Println("main start")
	keywords := []string{"web"}
	beginScrapping(keywords)
	fmt.Println("main end")
}

func beginScrapping(keywords []string) {
	fmt.Println("begin scrapping")
	ch := make(chan Job)
	var wg sync.WaitGroup
	wg.Add(1)
	go ScrapMeroJob(keywords, ch, &wg)
	wg.Add(1)
	go ScrapJobNepalWithChromium(keywords, &wg, ch)
	wg.Add(1)
	go ScrapJobejeeWithChromium(keywords, &wg, ch)
	go func(){
		wg.Wait()
		close(ch)
	}()

	for jobs := range ch {
		fmt.Printf("Job details from %v: title %v employer %v link %v \n", jobs.Source, jobs.Title, jobs.Employer, jobs.Link)
	}

	fmt.Println("end scrappin")
}

func ScrapJobNepalWithChromium(keywords []string, wg *sync.WaitGroup, ch chan Job) {
	defer (*wg).Done()
	url := fmt.Sprintf(`https://www.jobsnepal.com/search?q=%s`, strings.Join(keywords, "+"))
	ctx, cancel := chromedp.NewContext( 
		context.Background(), 
		chromedp.WithLogf(log.Printf), 
	) 
	defer cancel() 

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var nodes []*cdp.Node 
	chromedp.Run(timeoutCtx, 
		chromedp.Navigate(url), 
		chromedp.Nodes(".vb-content>div", &nodes, chromedp.ByQueryAll), 
	) 

	for _, node := range nodes {
		var newJob Job
		chromedp.Run(ctx, 
			chromedp.Text(".title", &newJob.Title, chromedp.ByQuery, chromedp.FromNode(node)),
			 chromedp.Text("h6>a", &newJob.Employer, chromedp.ByQuery, chromedp.FromNode(node)), 
			 chromedp.AttributeValue("h6>a", "href", &newJob.Link, nil, chromedp.ByQuery, chromedp.FromNode(node)),)

		newJob.Source = "Jobs Nepal"
		ch <- newJob
	}
}

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
				Link: domain + jobLink,
				Source: "Mero Job",
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

func ScrapJobejeeWithChromium(keywords []string, wg *sync.WaitGroup, ch chan Job) {
	defer (*wg).Done()
	ScrapAllJobejeePages(keywords, ch)
}

func ScrapAllJobejeePages(keywords []string, ch chan Job) {
	visitedPaginations := []int{1}
	prevUrl := ""
	url := fmt.Sprintf(`https://www.jobejee.com/job-search?q=%s`, strings.Join(keywords, "_"))

	ctx, cancel := chromedp.NewContext( 
		context.Background(), 
		chromedp.WithLogf(log.Printf), 
	) 
	defer cancel() 

	timeoutContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() 

	for prevUrl != url {
		prevUrl = url
		
		var nodes []*cdp.Node 

		chromedp.Run(ctx, 
			chromedp.Navigate(url), 
			chromedp.Nodes("div.lg-box", &nodes, chromedp.ByQueryAll), 
		) 

		for _, node := range nodes {
			var newJob Job
			chromedp.Run(ctx, 
				chromedp.Text("p.inner-header", &newJob.Title, chromedp.ByQuery, chromedp.FromNode(node)),
				chromedp.Text("div.client-name a", &newJob.Employer, chromedp.ByQuery, chromedp.FromNode(node)), 
				chromedp.AttributeValue("p.inner-header a", "href", &newJob.Link, nil, chromedp.ByQuery, chromedp.FromNode(node)),)
				newJob.Link = "https://www.jobejee.com" + newJob.Link
			newJob.Source = "Jobjee"
			ch <- newJob
		}

		var paginationNodes []*cdp.Node

		paginationErr := chromedp.Run(timeoutContext,chromedp.Nodes("ul.pagination>li", &paginationNodes, chromedp.ByQueryAll))
		if paginationErr == nil {
			newVisit := false
			for _, paginationNode := range paginationNodes {
				if newVisit {
					break
				}

				var pageN string
				chromedp.Run(ctx, chromedp.Text("a", &pageN, chromedp.ByQuery, chromedp.FromNode(paginationNode)))

				page, err := strconv.Atoi(pageN)

				if err == nil {
					alreadyVisited := false
					for _, visitedPage := range visitedPaginations {
						if visitedPage == page {
							alreadyVisited = true
							break
						}
					}

					if alreadyVisited {
						continue
					}

					newVisit = true

					visitedPaginations = append(visitedPaginations, page)

					url = fmt.Sprintf(`https://www.jobejee.com/job-search?q=%s&page=%d`, strings.Join(keywords, "_"), page)
				}
			}
		}
	}
}