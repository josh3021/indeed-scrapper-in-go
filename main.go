package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var baseURL string = "https://kr.indeed.com/jobs?q=python&limit=50"

type extractedJob struct {
	id       string
	title    string
	company  string
	location string
	salary   string
	summary  string
}

func main() {
	var jobs []extractedJob
	c := make(chan []extractedJob)
	totalPages := getPages()

	for i := 0; i < totalPages; i++ {
		go getPage(i, c)
	}

	for i := 0; i < totalPages; i++ {
		extractedJobs := <-c
		jobs = append(jobs, extractedJobs...)
	}

	writeJobs(jobs)
	log.Println("Done, extracted", len(jobs))
}

func getPage(page int, c chan<- []extractedJob) {
	var jobs []extractedJob
	extractJobChannel := make(chan extractedJob)
	pageURL := baseURL + "&start=" + strconv.Itoa(page*50)
	log.Println("Requesting", pageURL)
	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	searchCards := doc.Find(".jobsearch-SerpJobCard")

	searchCards.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, extractJobChannel)
	})

	for i := 0; i < searchCards.Length(); i++ {
		job := <-extractJobChannel
		jobs = append(jobs, job)
	}
	c <- jobs
}

func getPages() int {
	pages := 0
	res, err := http.Get(baseURL)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})

	return pages
}

func extractJob(card *goquery.Selection, c chan<- extractedJob) {
	id, _ := card.Attr("data-jk")
	title := cleanString(card.Find(".title>a").Text())
	company := cleanString(card.Find(".company").Text())
	location := cleanString(card.Find(".sjcl").Text())
	salary := cleanString(card.Find(".salaryText").Text())
	summary := cleanString(card.Find(".summary").Text())

	c <- extractedJob{
		id:       id,
		title:    title,
		company:  company,
		location: location,
		salary:   salary,
		summary:  summary,
	}
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv")
	writeChan := make(chan []string)
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"Link", "Title", "Location", "Salary", "Summary"}

	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs {
		go writeJob(job, writeChan)
	}

	for i := 0; i < len(jobs); i++ {
		job := <-writeChan
		jwErr := w.Write(job)
		checkErr(jwErr)
	}

}

func writeJob(job extractedJob, c chan<- []string) {
	jobSlice := []string{"https://kr.indeed.com/viewjob?jk=" + job.id, job.title, job.location, job.salary, job.summary}
	c <- jobSlice
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Response Failed Code:", res.StatusCode)
	}
}

func cleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}
