package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

const (
	URL = "http://api.marketstack.com/v1/intraday/latest?access_key=cd43ead81496e2c9c20ddc9178b71a74&symbols="
	DescURL = "https://www.marketwatch.com/investing/stock/"
)



func main() {
	app := app.New()
	win := app.NewWindow("Tickers dashboard")
	win.Resize(fyne.NewSize(600, 400))

	var data  = struct {
		Tickers []string
		Amounts []string
	}{
		Tickers: []string{"AAPL", "TSLA"},
	}

	tickers := binding.BindStringList(&data.Tickers)
	amounts := binding.BindStringList(&data.Amounts)


	addEntry := widget.NewEntry()
	addButton := widget.NewButton("add", func() {
		data.Tickers = append(data.Tickers, addEntry.Text)
		data.Amounts = append(data.Amounts, "0")
		addEntry.Text = ""
		addEntry.Refresh()
		err := tickers.Reload()
		if err != nil {
			log.Println(err)
		}
		err = amounts.Reload()
		if err != nil {
			log.Println(err)
		}
	})

	tickersList := widget.NewListWithData(tickers,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id binding.DataItem, object fyne.CanvasObject) {
			object.(*widget.Label).Bind(id.(binding.String))
		})

	amountsList := widget.NewListWithData(amounts,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id binding.DataItem, object fyne.CanvasObject) {
			object.(*widget.Label).Bind(id.(binding.String))
		})

	go func() {
		for {
			var result = make([]string, len(data.Tickers))
			volumes, err := GetAmounts(data.Tickers)
			if err != nil {
				log.Println(err)
			}
			for i, volume := range volumes {
				result[i] = volume
			}
			data.Amounts = result
			log.Println(result)
			log.Println(data)
			err = amounts.Reload()
			if err != nil {
				log.Println(err)
			}
			amountsList.Refresh()
			time.Sleep(time.Second * 10)
		}
	}()

	descriptionWidget := widget.NewLabel("")

	tickersList.OnSelected = func(id widget.ListItemID) {
		desc, err := GetDescription(data.Tickers[id])
		if err != nil {
			log.Println(err)
		}
		log.Println(desc)
		descriptionWidget.SetText(desc)
	}

	addContainer := container.NewVBox(addEntry, addButton)
	rightCont := container.NewBorder(descriptionWidget, addContainer, nil, nil)
	leftCont := container.NewGridWithColumns(2, tickersList, amountsList)

	win.SetContent(
		container.NewHSplit(leftCont, rightCont),
	)
	win.ShowAndRun()
}


func GetAmounts (tickers []string) ([]string, error) {

	var amounts = make([]string, 0)
	url := URL
	for _,ticker := range tickers {
		url = fmt.Sprintf("%s%s,", url, ticker)
	}
	client := http.Client{
		Timeout:       time.Second*30,
	}
	request, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return amounts, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return amounts, errors.New("connection error")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return amounts, err
	}

	type Result struct {
		Date string `json:"date"`
		Symbol string `json:"symbol"`
		Exchange string `json:"exchange"`
		Open float32 `json:"open"`
		High float32 `json:"high"`
		Low float32 `json:"low"`
		Close float32 `json:"close"`
		Last float32 `json:"last"`
		Volume float32 `json:"volume"`
	}
	type Quote struct {
		Data []Result `json:"data"`
	}
	var result Quote
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Println(err)
		return amounts, err
	}
	log.Printf("%+v", result)
	for _, position := range result.Data {
		amounts = append(amounts, fmt.Sprintf("%f", position.Volume))
	}
	//log.Printf("tickers: %v", tickers)
	//var xxx = make([]string, 0)
	//for range tickers {
	//	xxx = append(xxx, fmt.Sprintf("%d", time.Now().Unix()))
	//}
	//return amounts, nil
	return []string{"170.09", "1025.49"}, nil
}

func GetDescription (ticker string) (string, error) {
	client := http.Client{
		Timeout: time.Second*30,
	}
	url := fmt.Sprintf("%s%s/company-profile?mod=mw_quote_tab", DescURL, ticker)
	response, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	dataInString := string(data)

	re := regexp.MustCompile(`(<p class="description__text">)(.+)(</p>)`)
	parts := re.FindStringSubmatch(dataInString)
	if parts == nil {
		return "", errors.New("no description")
	}

	return parts[2], nil
}
