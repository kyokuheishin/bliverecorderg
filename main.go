package main

import (
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
)

var roomId = flag.Int("roomid", -1, "The id of the room which you want to record.")

const bilibiliRoomApiUri = "https://api.live.bilibili.com/room/v1/Room/room_init"
const bilibiliLiveRealStreamUri = "https://api.live.bilibili.com/xlive/web-room/v1/playUrl/playUrl"

func main() {
	flag.Parse()

	if *roomId == -1 {
		fmt.Println("Please input a valid room id.")
		return
	}
	// fmt.Println("room id.")

	request, err := http.NewRequest("GET", bilibiliRoomApiUri, nil)

	if err != nil {
		log.Fatal(err)
	}

	params := request.URL.Query()
	params.Add("id", strconv.Itoa(*roomId))
	request.URL.RawQuery = params.Encode()

	timeout := time.Duration(5 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(body))
	if err != nil {
		log.Fatal(err)
	}

	respJson, err := simplejson.NewJson(body)

	if err != nil {
		log.Fatal(err)
	}

	code := respJson.Get("code").MustInt()
	fmt.Println(code)
	if code != 0 {
		fmt.Printf("Unknown error. Maybe there are some issues of Bilibili's server")
		return
	}

	liveStatus := respJson.GetPath("data", "live_status").MustInt()
	fmt.Println(liveStatus)
	if liveStatus == 0 {
		fmt.Printf("Not living.")
		return
	}

	realRoomId := respJson.GetPath("data", "room_id").MustInt()
	fmt.Println(realRoomId)
	streamUrlRequest, err := http.NewRequest("GET", bilibiliLiveRealStreamUri, nil)

	if err != nil {
		log.Fatal(err)
	}

	realRoomIdStr := strconv.Itoa(realRoomId)

	params = streamUrlRequest.URL.Query()
	params.Add("cid", realRoomIdStr)
	params.Add("qn", strconv.Itoa(10000))
	params.Add("platform", "web")
	params.Add("https_url_req", "1")
	params.Add("ptype", "16")

	streamUrlRequest.URL.RawQuery = params.Encode()

	streamUrlResponse, err := client.Do(streamUrlRequest)
	if err != nil {
		log.Fatal(err)
	}

	defer streamUrlResponse.Body.Close()

	streamUrlBody, err := ioutil.ReadAll(streamUrlResponse.Body)
	// fmt.Println(string(streamUrlBody))
	if err != nil {
		log.Fatal(err)
	}

	streamUrlJson, err := simplejson.NewJson(streamUrlBody)

	if err != nil {
		log.Fatal(err)
	}

	realStreamUrl := streamUrlJson.GetPath("data", "durl").GetIndex(0).Get("url").MustString()

	fmt.Println(realStreamUrl)

	err = os.MkdirAll(realRoomIdStr, os.ModePerm)

	if err != nil {
		log.Fatal(err)
	}
	nowTime := time.Now()
	formattedNowTimeStr := nowTime.Format("2006-01-02-15-04-05")
	out, err := os.Create(realRoomIdStr + "/" + realRoomIdStr + "_" + formattedNowTimeStr + ".flv")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	fileResp, err := http.Get(realStreamUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer fileResp.Body.Close()

	fmt.Println("Start downloading...")

	_, err = io.Copy(out, fileResp.Body)

	if err != nil {
		log.Fatal(err)
	}

}
