package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/Resousse/automatic-mouse-mover/pkg/mousemover"
	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	"github.com/kirsle/configdir"
	log "github.com/sirupsen/logrus"
)

type AppSettings struct {
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

var configPath = configdir.LocalConfig("amm")
var configFile = filepath.Join(configPath, "settings.json")

const alphaInactive = 0.6

var (
	colorBlue  = color.RGBA{30, 144, 255, 255}
	colorRed   = color.RGBA{255, 0, 0, 255}
	colorWhite = color.RGBA{255, 255, 255, 255}
)

func main() {
	systray.Run(onReady, onExit)
}

func loadIconFile(iconName string) []byte {
	if iconName != "mouse" && iconName != "cloud" && iconName != "geometric" && iconName != "man" {
		iconName = "mouse"
	}
	iconPaths := []string{}
	if base := os.Getenv("AMM_ICON_DIR"); base != "" {
		iconPaths = append(iconPaths, filepath.Join(base, "assets", "icon", iconName+".png"))
	}
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	iconPaths = append(iconPaths,
		exPath+"/../Resources/assets/icon/"+iconName+".png",
		exPath+"/../assets/icon/"+iconName+".png",
		"./assets/icon/"+iconName+".png",
	)

	for _, iconPath := range iconPaths {
		if _, err := os.Stat(iconPath); err == nil {
			b, err := os.ReadFile(iconPath)
			if err == nil && len(b) > 0 {
				return b
			}
		}
	}
	panic("Failed to load icon: " + iconName + ".png")
}

func getMenuIcon(iconName string) []byte {
	return loadIconFile(iconName)
}

func getTrayIcon(iconName string, active bool, col string) []byte {
	b := loadIconFile(iconName)

	if active && col != "" {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			log.Fatalln(err)
		}
		var dimg *image.RGBA = image.NewRGBA(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				if a != 0 {
					switch col {
					case "white":
						dimg.Set(x, y, colorWhite)
					case "red":
						dimg.Set(x, y, colorRed)
					default:
						dimg.Set(x, y, colorBlue)
					}
				} else {
					dimg.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
				}
			}
		}
		var c bytes.Buffer
		png.Encode(&c, dimg)
		return c.Bytes()
	}

	if !active {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			log.Fatalln(err)
		}
		var dimg *image.RGBA = image.NewRGBA(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				newAlpha := uint8(float64(a>>8) * alphaInactive)
				dimg.Set(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), newAlpha})
			}
		}
		var c bytes.Buffer
		png.Encode(&c, dimg)
		return c.Bytes()
	}

	return b
}

func setIcon(iconName string, color string, configFile string, settings *AppSettings, active ...bool) {
	isActive := len(active) != 0 && active[0]
	var iconData []byte
	if isActive && color != "" {
		iconData = getTrayIcon(iconName, true, color)
		systray.SetIcon(iconData)
	} else {
		iconData = getTrayIcon(iconName, isActive, "")
		systray.SetTemplateIcon(iconData, iconData)
	}
	if configFile != "" {
		*settings = AppSettings{iconName, color}
		fh, err := os.Create(configFile)
		if err != nil {
			log.Errorf("Failed to create config file: %v", err)
			return
		}
		defer fh.Close()
		encoder := json.NewEncoder(fh)
		encoder.Encode(settings)
	}
}

func onReady() {
	go func() {
		err := configdir.MakePath(configPath)
		if err != nil {
			panic(err)
		}
		var settings AppSettings
		settings = AppSettings{"mouse", "blue"}

		if _, err = os.Stat(configFile); os.IsNotExist(err) {
			fh, err := os.Create(configFile)
			if err != nil {
				panic(err)
			}
			defer fh.Close()
			encoder := json.NewEncoder(fh)
			encoder.Encode(settings)

		} else {
			fh, err := os.Open(configFile)
			if err != nil {
				panic(err)
			}
			defer fh.Close()

			decoder := json.NewDecoder(fh)
			decoder.Decode(&settings)
		}

		about := systray.AddMenuItem("About AMM", "Information about the app")
		systray.AddSeparator()
		ammStart := systray.AddMenuItem("Start", "start the app")
		ammStop := systray.AddMenuItem("Stop", "stop the app")
		timeLeft := systray.AddMenuItem("", "Time left before the app stops")
		timeLeft.Disable()
		timeLeft.Hide()

		durationMenu := systray.AddMenuItem("Duration", "how long the app keeps running")
		durationItems := make([]*systray.MenuItem, len(durationOptions))
		durationClicks := make(chan int)
		for i, option := range durationOptions {
			durationItems[i] = durationMenu.AddSubMenuItemCheckbox(option.label, option.label, i == 0)
			go func() {
				for range durationItems[i].ClickedCh {
					durationClicks <- i
				}
			}()
		}

		icons := systray.AddMenuItem("Icons", "icon of the app")
		mouse := icons.AddSubMenuItem("Mouse", "Mouse icon")
		mouseIcon := getMenuIcon("mouse")
		mouse.SetTemplateIcon(mouseIcon, mouseIcon)
		cloud := icons.AddSubMenuItem("Cloud", "Cloud icon")
		cloudIcon := getMenuIcon("cloud")
		cloud.SetTemplateIcon(cloudIcon, cloudIcon)
		man := icons.AddSubMenuItem("Man", "Man icon")
		manIcon := getMenuIcon("man")
		man.SetTemplateIcon(manIcon, manIcon)
		geometric := icons.AddSubMenuItem("Geometric", "Geometric")
		geometricIcon := getMenuIcon("geometric")
		geometric.SetTemplateIcon(geometricIcon, geometricIcon)

		colors := systray.AddMenuItem("Icon Colors", "")
		system := colors.AddSubMenuItem("System", "System default color")
		blue := colors.AddSubMenuItem("Blue 🔵", "Blue")
		white := colors.AddSubMenuItem("White ⚪️", "White")
		red := colors.AddSubMenuItem("Red 🔴", "Red")

		ammStop.Disable()
		setIcon(settings.Icon, settings.Color, "", &settings, true)
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
		// Sets the icon of a menu item. Only available on Mac.
		//mQuit.SetIcon(icon.Data)
		mouseMover := mousemover.GetInstance()
		mouseMover.Start()
		ammStart.Disable()
		ammStop.Enable()

		selected := 0
		var cd countdown
		shownLeft := ""
		// showRemaining updates the Time left entry when the text changes.
		showRemaining := func() {
			text := ""
			if cd.active() {
				text = formatRemaining(cd.remaining(time.Now()))
			}
			if text == shownLeft {
				return
			}
			shownLeft = text
			if text == "" {
				timeLeft.Hide()
			} else {
				timeLeft.SetTitle("Time left: " + text)
				timeLeft.Show()
			}
		}
		startMover := func() {
			log.Infof("starting the app")
			mouseMover.Start()
			ammStart.Disable()
			ammStop.Enable()
			setIcon(settings.Icon, settings.Color, configFile, &settings, true)
			cd.start(durationOptions[selected].length, time.Now())
			showRemaining()
		}
		stopMover := func() {
			log.Infof("stopping the app")
			ammStart.Enable()
			ammStop.Disable()
			mouseMover.Quit()
			setIcon(settings.Icon, settings.Color, configFile, &settings, false)
			cd.stop()
			showRemaining()
		}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ammStart.ClickedCh:
				startMover()

			case <-ammStop.ClickedCh:
				stopMover()

			case i := <-durationClicks:
				durationItems[selected].Uncheck()
				selected = i
				durationItems[selected].Check()
				if ammStart.Disabled() {
					cd.start(durationOptions[selected].length, time.Now())
					showRemaining()
				} else if durationOptions[selected].length > 0 {
					startMover()
				}

			case <-ticker.C:
				if cd.active() && cd.remaining(time.Now()) <= 0 {
					log.Infof("working duration is over")
					stopMover()
				} else {
					showRemaining()
				}

			case <-mQuit.ClickedCh:
				log.Infof("Requesting quit")
				mouseMover.Quit()
				systray.Quit()
				return
			case <-mouse.ClickedCh:
				setIcon("mouse", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-cloud.ClickedCh:
				setIcon("cloud", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-man.ClickedCh:
				setIcon("man", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-geometric.ClickedCh:
				setIcon("geometric", settings.Color, configFile, &settings, ammStart.Disabled())
			case <-system.ClickedCh:
				setIcon(settings.Icon, "", configFile, &settings, ammStart.Disabled())
			case <-blue.ClickedCh:
				setIcon(settings.Icon, "blue", configFile, &settings, ammStart.Disabled())
			case <-red.ClickedCh:
				setIcon(settings.Icon, "red", configFile, &settings, ammStart.Disabled())
			case <-white.ClickedCh:
				setIcon(settings.Icon, "white", configFile, &settings, ammStart.Disabled())
			case <-about.ClickedCh:
				log.Infof("Requesting about")
				robotgo.Alert("Automatic-mouse-mover app v1.4", "Created by Prashant Gupta. \n\nMore info at: https://github.com/resousse/automatic-mouse-mover", "OK", "")
			}
		}

	}()
}

func onExit() {
	// clean up here
	log.Infof("Finished quitting")
}
