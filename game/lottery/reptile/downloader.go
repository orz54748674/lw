package reptile

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"vn/storage/lotteryStorage"

	"github.com/PuerkitoBio/goquery"
)

type Downloader struct {
	client *http.Client
	url    string
	title  string
	area   string
}

func NewDownLoader(url, title, area string) (d *Downloader, err error) {
	d = &Downloader{
		url:   url,
		title: title,
		area:  area,
	}
	d.client, err = d.getHttpClient()
	return
}

func (d *Downloader) Download(url string) (openCode map[lotteryStorage.PrizeLevel][]string, number string, err error) {
	d.url = url
	req, _ := http.NewRequest("GET", d.url, nil)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36")

	//resp, err := d.client.Get(d.url)
	resp, err := d.client.Do(req)
	if err != nil {
		return
	}
	openCode = make(map[lotteryStorage.PrizeLevel][]string)
	number, err = d.parse(resp.Body, openCode)
	return
}

func (d *Downloader) parse(data io.ReadCloser, codes map[lotteryStorage.PrizeLevel][]string) (number string, err error) {

	defer data.Close()
	html, err := goquery.NewDocumentFromReader(data)
	if err != nil {
		return
	}

	table := html.Find(".box_kqxs").First()
	if len(table.Text()) == 0 {
		err = fmt.Errorf("not find data")
		return
	}
	dateInfo := strings.TrimSpace(table.Find("div.ngay").Text())
	number = dateInfo[len(dateInfo)-10:]
	numbers := strings.Split(number, "/")
	if len(numbers) != 3 {
		err = fmt.Errorf("number err:%v;numbers:%v", number, numbers)
		return
	}
	number = strings.Join(numbers, "-")
	levels := []string{"db", "1", "2", "3", "4", "5", "6", "7", "8"}
	count := 0
	for i := 0; i < len(levels); i++ {
		st := fmt.Sprintf(".box_kqxs_content td.giai%s div", levels[i])
		table.Find(st).Each(func(_ int, item *goquery.Selection) {
			key := lotteryStorage.PrizeLevel(fmt.Sprintf("L%d", i))
			codes[key] = append(codes[key], strings.TrimSpace(item.Text()))
		})
		if d.area == "North" && i == 8 {
			break
		}
		count++
	}
	return
}

func (d *Downloader) getHttpClient() (client *http.Client, err error) {
	if len(d.url) == 0 {
		return nil, fmt.Errorf("collect url is empty")
	}
	proxy, _ := url.Parse("http://wjlyf2000:J9xpTOLIGflyXNqZ@proxy.packetstream.io:31112")

	if strings.HasPrefix(d.url, "http://") {
		tr := &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
		return &http.Client{Transport: tr, Timeout: 10 * time.Second}, nil
	} else if strings.HasPrefix(d.url, "https://") {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyURL(proxy),
		}
		return &http.Client{Transport: tr, Timeout: 10 * time.Second}, nil
	} else {
		return nil, fmt.Errorf("protocol error")
	}
}
