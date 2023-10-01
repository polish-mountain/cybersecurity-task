package main

import (
	"log"

	"github.com/go-rod/rod"
	"github.com/google/uuid"
)

const SCREENSHOT_WORKERS_NUM = 1

type ScreenshotJob struct {
	URL      string
	HostInfo *HostInfo
}

var screenshotJobQueue = make(chan ScreenshotJob, 1000)

func startScreenshotWorkers() {
	for i := 0; i < SCREENSHOT_WORKERS_NUM; i++ {
		go runScreenshotWorker()
	}
}

func runScreenshotWorker() {

	for job := range screenshotJobQueue {
		err := processJob(job)
		if err != nil {
			log.Printf("Failed to screenshot %v: %v", job.URL, err)
		}
	}

}

func addHostScreenshot(host *HostInfo, uuid string, title string) {
	resultUrl := *rootURL + "/api/screenshot/" + uuid
	for _, v := range host.Screenshots {
		if v == resultUrl {
			return
		}
	}
	log.Printf("adding screenshot to array %v", host.IP)
	host.Screenshots = append(host.Screenshots, resultUrl)
	if len(host.OpenServices) > 0 {
		host.OpenServices[0].Title = title
	}
	updateHost(host)

}

func processJob(job ScreenshotJob) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// check if screenshot already exists
	var cs CachedScreenshot
	err = db.Where("url = ?", job.URL).First(&cs).Error
	if err == nil {
		addHostScreenshot(job.HostInfo, cs.UUID, cs.Title)
		log.Printf("ADDING CACHED SCREENSHOT FOR %v", job.URL)
		return nil
	}

	page := rod.New().MustConnect().MustPage(job.URL)
	defer page.MustClose()
	page.MustWaitStable()
	page.MustScreenshot()
	title := page.MustElement("title").MustEval(`() => this.innerText`).String()
	screenshotData, err := page.Screenshot(false, nil)
	if err != nil {
		return err
	}
	newCs := &CachedScreenshot{
		URL:   job.URL,
		UUID:  uuid.New().String(),
		Data:  screenshotData,
		Title: title,
	}
	err = db.Save(newCs).Error
	if err != nil {
		return err
	}
	addHostScreenshot(job.HostInfo, newCs.UUID, title)
	return nil
}
