package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/effects"
	"github.com/gdamore/tcell/v2"
	"github.com/tidwall/gjson"
)
const (
	AppName         = "GoWave"
	Instructions    = "space or \"P\" for play/pause UP & DOWN key for change volume and \"Esc\" for exit"
	InitialStatus   = "Waiting for channel selection ..."
	VolumeIncrement = 0.05
)

var (
	Keys = []rune("1234567890qwertyuiop")
)

var (
	urls       []map[string]string
	isPlaying  bool
	playingNow int = -1
	volume     float64 = 0.5
	status     string
	mu         sync.Mutex
	streamer   beep.StreamSeekCloser
	volumeCtrl *effects.Volume
	ctrl       *beep.Ctrl
)

func loadData() error {
	resp, err := http.Get("https://radio.9craft.ir/v1/api/genre/all")
	if err != nil {
		return fmt.Errorf("error fetching radio stations: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	data := gjson.Get(string(body), "data")
	if data.Exists() {
		mu.Lock()
		urls = make([]map[string]string, 0)
		data.ForEach(func(key, value gjson.Result) bool {
			url := make(map[string]string)
			value.ForEach(func(k, v gjson.Result) bool {
				url[k.String()] = v.String()
				return true
			})
			urls = append(urls, url)
			return true
		})
		mu.Unlock()
	}
	return nil
}

func initScreen() tcell.Screen {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack))
	s.Clear()

	return s
}

func updateDisplay(s tcell.Screen, status string) {
	s.Clear()
	row := 0
	drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorGreen), AppName)
	row += 2

	drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorBlue), "Status: ")
	drawText(s, 8, row, tcell.StyleDefault.Foreground(tcell.ColorPurple), status)
	row += 2

	mu.Lock()
	defer mu.Unlock()

	if playingNow != -1 {
		drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorYellow), "Genre: ")
		drawText(s, 7, row, tcell.StyleDefault.Foreground(tcell.ColorBlue), urls[playingNow]["server_name"])
		row++

		drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorYellow), "Music: ")
		drawText(s, 7, row, tcell.StyleDefault.Foreground(tcell.ColorBlue), urls[playingNow]["title"])
		row++
	}

	row++

	for i, url := range urls {
		prefix := fmt.Sprintf("%c. ", Keys[i])
		if i == playingNow {
			drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorGreen), prefix)
			drawText(s, len(prefix), row, tcell.StyleDefault.Foreground(tcell.ColorGreen), url["server_name"]+": ")
		} else {
			drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorBlue), prefix)
			drawText(s, len(prefix), row, tcell.StyleDefault.Foreground(tcell.ColorBlue), url["server_name"]+": ")
		}
		drawText(s, len(prefix)+len(url["server_name"])+2, row, tcell.StyleDefault.Foreground(tcell.ColorWhite), url["title"])
		row++
	}

	row++
	drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorGray), Instructions)
	row += 2

	drawText(s, 0, row, tcell.StyleDefault.Foreground(tcell.ColorWhite), fmt.Sprintf("Volume: %.0f%%", volume*100))

	s.Show()
}


func drawText(s tcell.Screen, x, y int, style tcell.Style, text string) {
	for _, r := range []rune(text) {
		s.SetContent(x, y, r, nil, style)
		x++
	}
}


func handleVolumeChange(change float64) {
	volume += change
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	if volumeCtrl != nil {
		speaker.Lock()
		volumeCtrl.Volume = -1 + 2*volume 
		speaker.Unlock()
	}
}


func main() {
	fmt.Println("Loading stations ...")
	if err := loadData(); err != nil {
		log.Fatalf("Error loading radio stations: %v", err)
	}

	s := initScreen()
	defer s.Fini()

	status = InitialStatus
	updateDisplay(s, status)

	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-time.After(10 * time.Second):
				if err := loadData(); err != nil {
					status = fmt.Sprintf("Error: %v", err)
				} else {
					status = InitialStatus
				}
				updateDisplay(s, status)
			case <-quit:
				return
			}
		}
	}()

	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				close(quit)
				return
			case tcell.KeyUp:
				handleVolumeChange(VolumeIncrement)
			case tcell.KeyDown:
				handleVolumeChange(-VolumeIncrement)
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'p', 'P', ' ':
					if playingNow != -1 {
						if isPlaying {
							speaker.Lock()
							ctrl.Paused = true
							speaker.Unlock()
							status = "Paused"
						} else {
							speaker.Lock()
							ctrl.Paused = false
							speaker.Unlock()
							status = "Playing..."
						}
						isPlaying = !isPlaying
					}
				default:
					index := strings.IndexRune(string(Keys), ev.Rune())
					if index != -1 && index < len(urls) {
						if streamer != nil {
							streamer.Close()
						}
						resp, err := http.Get(urls[index]["http_server_url"])
						if err != nil {
							log.Printf("Error loading stream: %v", err)
							continue
						}
						streamer, format, err := mp3.Decode(resp.Body)
						if err != nil {

							status = "Error: Unsupported stream format"
							updateDisplay(s, status)
							continue
						}
						volumeCtrl = &effects.Volume{
							Streamer: streamer,
							Base:     2, 
							Volume:   -1 + 2*volume,
						}
						ctrl = &beep.Ctrl{Streamer: volumeCtrl, Paused: false}
						speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
						speaker.Play(ctrl)
						playingNow = index
						isPlaying = true
						status = "Playing..."
					}
				}
			}
		}
		updateDisplay(s, status)
	}
}
