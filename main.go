package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/robfig/cron/v3"
)

var roomId = flag.Int("roomid", -1, "The id of the room which you want to record.")

var roomChecklist = make(chan int)

const bilibiliRoomApiUri = "https://api.live.bilibili.com/room/v1/Room/room_init"
const bilibiliLiveRealStreamUri = "https://api.live.bilibili.com/xlive/web-room/v1/playUrl/playUrl"

func checkLiveStatus(roomId int) (realRoomId int, liveStatus bool, err error) {
	request, err := http.NewRequest("GET", bilibiliRoomApiUri, nil)

	params := request.URL.Query()
	params.Add("id", strconv.Itoa(roomId))
	request.URL.RawQuery = params.Encode()

	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	response, err := client.Do(request)

	if err != nil {
		return -1, false, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(body))

	respJson, err := simplejson.NewJson(body)

	code := respJson.Get("code").MustInt()
	fmt.Println(code)
	if code != 0 {

		return -1, false, errors.New("Unknown error. Maybe there are some issues of Bilibili's server")
	}
	fmt.Println(string(body))
	liveStatus = respJson.GetPath("data", "live_status").MustInt() != 0

	realRoomId = respJson.GetPath("data", "room_id").MustInt()

	return realRoomId, liveStatus, err
}

func liveVideoDownloader(realRoomId int) {

	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	streamUrlRequest, err := http.NewRequest("GET", bilibiliLiveRealStreamUri, nil)

	if err != nil {
		log.Default().Printf(err.Error())
		return
	}

	realRoomIdStr := strconv.Itoa(realRoomId)

	params := streamUrlRequest.URL.Query()
	params.Add("cid", realRoomIdStr)
	params.Add("qn", strconv.Itoa(10000))
	params.Add("platform", "web")
	params.Add("https_url_req", "1")
	params.Add("ptype", "16")

	streamUrlRequest.URL.RawQuery = params.Encode()

	streamUrlResponse, err := client.Do(streamUrlRequest)
	if err != nil {
		log.Default().Printf(err.Error())
		return
	}

	defer streamUrlResponse.Body.Close()

	streamUrlBody, err := ioutil.ReadAll(streamUrlResponse.Body)

	if err != nil {
		log.Default().Printf(err.Error())
		return
	}

	streamUrlJson, err := simplejson.NewJson(streamUrlBody)

	if err != nil {
		log.Default().Printf(err.Error())
		return
	}

	realStreamUrl := streamUrlJson.GetPath("data", "durl").GetIndex(0).Get("url").MustString()

	fmt.Println(realStreamUrl)

	err = os.MkdirAll(realRoomIdStr, os.ModePerm)

	if err != nil {
		log.Default().Printf(err.Error())
		return
	}
	nowTime := time.Now()
	formattedNowTimeStr := nowTime.Format("2006-01-02-15-04-05")
	out, err := os.Create(realRoomIdStr + "/" + realRoomIdStr + "_" + formattedNowTimeStr + ".flv")
	if err != nil {
		log.Default().Printf(err.Error())
		return
	}
	defer out.Close()

	fileResp, err := http.Get(realStreamUrl)
	if err != nil {
		log.Default().Printf(err.Error())
		return
	}

	defer fileResp.Body.Close()

	log.Default().Println("Real Room Id:" + realRoomIdStr + "Start downloading at" + formattedNowTimeStr)

	_, err = io.Copy(out, fileResp.Body)

	if err != nil {
		log.Default().Printf(err.Error())
		return
	}
}

func main() {

	c := cron.New()

	flag.Parse()

	if *roomId == -1 {
		fmt.Println("Please input a valid room id.")
		return
	}

	c.AddFunc("@every 2s", func() {
		fmt.Println("hi")
		roomIdNeedToBeCheck := <-roomChecklist
		fmt.Println("Now checking:" + strconv.Itoa(roomIdNeedToBeCheck))
		defer func() { roomChecklist <- roomIdNeedToBeCheck }()
		realRoomId, status, err := checkLiveStatus(roomIdNeedToBeCheck)

		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if !status {
			fmt.Println("Not living...")
			return
		}

		liveVideoDownloader(realRoomId)

	})

	c.Start()

	roomChecklist <- *roomId

	select {}

}
