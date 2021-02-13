// Copyright 2020 DERO Foundation. All rights reserved.
// build win: -ldflags -H=windowsgui
// TODO: Code cleanup, seed languages, multi-language support, daemon/miner integration.

package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/deroproject/derosuite/address"
	"github.com/deroproject/derosuite/crypto"
	"github.com/deroproject/derosuite/globals"
	"github.com/deroproject/derosuite/transaction"
	"github.com/deroproject/derosuite/walletapi"
	rl "github.com/DankFC/raylib-goplus/raylib"
	"github.com/blang/semver"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type _Keys struct {
	Spendkey_Secret crypto.Key `json:"spendkey_secret"`
	Spendkey_Public crypto.Key `json:"spendkey_public"`
	Viewkey_Secret  crypto.Key `json:"viewkey_secret"`
	Viewkey_Public  crypto.Key `json:"viewkey_public"`
}

type Config struct {
	RemoteDaemon		string `json:"RemoteDaemon"`
	RemoteDaemonTestnet	string `json:"RemoteDaemonTestnet"`
	LocalDaemon			string `json:"LocalDaemon"`
	LocalDaemonTestnet	string `json:"LocalDaemonTestnet"`
	RPCAuth				string `json:"RPCAuth"`
	RPCAddress			string `json:"RPCAddress"`
	DefaultMode			string `json:"DefaultMode"`
	DefaultNetwork		string `json:"DefaultNetwork"`
}

type Session struct {
	Path		string
	Mode		string // network mode (remote, local, offline)
	Syncing		bool
	Network		string
	Daemon		string
	Rescan		bool
	RPCServer	bool
	RPCAuth		string
	RPCAddress	string
	Color		rl.Color
}

type Transfer struct {
	rAddress	*address.Address
	PaymentID	[]byte
	Amount		uint64
	Fees		uint64
	TX			*transaction.Transaction
	TXID		crypto.Hash
	Size		float32
	Status		string
	Inputs		[]uint64
	InputSum	uint64
	Change		uint64
	Relay		bool
	OfflineTX	bool
	Filename	string
}

type Record struct {
	txNumber		uint64
	txTime			string
	txHeight		string
	txTopoHeight	string
	txID			string
	txPaymentID		string
	txAmount		string
	txKey			string
	txColor			rl.Color
}

// Constants
const windowWidth = 1600
const windowHeight = 800
const TARGET_FPS = 60
const APP_PATH = ""
const ASSET_PATH = "assets/"
const FONT_PATH = "assets/fonts/"
const DEFAULT_REMOTE_NODE = "http://rwallet.dero.live:20206"
const DEFAULT_LOCAL_NODE = "http://127.0.0.1:20206"
const DEFAULT_REMOTE_NODE_TESTNET = "http://explorer.dero.io:30306"
const DEFAULT_LOCAL_NODE_TESTNET = "http://127.0.0.1:30306"
const DEFAULT_RPC_ADDRESS = "127.0.0.1:20209"
const DEFAULT_RPC_ADDRESS_TESTNET = "127.0.0.1:30309"
const MAX_PW_LENGTH = 30

// Globals
var Version = semver.MustParse("0.3.0-3.Alpha")
var config Config
var wallet *walletapi.Wallet
var account = &walletapi.Account{} // all account  data is available here
var session Session
var transfer Transfer

var daemonOnline bool 
var start int = 0
var wHeight uint64
var dHeight uint64
var networkStatus bool
var isFullscreen bool = false
var drawFPS = false
var spacing = float32(0)
var lineSpacing = float32(0)
var progressBar rl.Rectangle
var progressTotal int
var progressNow float32 = 0
var fontSize float32 = 20
var statusBar rl.Rectangle
var command string
var em bool
var sidebar bool = false
var sidebarText string = ""
var createAccountCompleted bool = false
var seed string
var address_s string
var statusText string = "Ready"
var walletText string = ""
var walletTextColor rl.Color = rl.White
var statusBarX float32 = 0
var statusBarY float32
var statusBarWidth float32
var statusColor rl.Color = rl.White
var currentTime = time.Now()
var currentTimeWidth rl.Vector2
var windowIndex float32 = 0
var availableText string = ""
var pendingText string = ""
var createAccountFilename string = ""
var createAccountFilenameEditable bool = false
var createAccountPassword string = ""
var createAccountPasswordEditable bool = false
var currentAccountPassword string
var tempPasswordEditable bool = true
var tempPasswordUpdated bool = false
var tempPassword string = ""
var files string
var dropboxEdit bool = false
var dropboxActive int = 0
var dropboxOld int = 0
var loginPassword string = ""
var loginPasswordEditable bool = true
var restorePassword string = ""
var restorePasswordEditable bool = true
var restoreFilename string = ""
var restoreFilenameEditable bool = true
var restoreHex string = ""
var rescanHeight uint64
var rescanPath string
var rescanString string
var amountString string
var pidString string
var receiverString string
var rescanEditable bool = true
var rescanUpdated bool = false
var fileError bool = false
var passwordError bool = false
var restoreHexEditable bool = true
var finalHex string
var err error
var src rl.Rectangle
var dst rl.Rectangle
var active int
var checked bool = false
var cmdGray rl.Color
var cmdBlue rl.Color
var cmdGreen rl.Color
var transparent rl.Color

// Main program loop
func main() {
	// Set up logger
	log_file, err := os.OpenFile(APP_PATH + "logs.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	
    if err != nil {
        log.Fatal(err)
    }

    defer log_file.Close()
	
	log.SetOutput(log_file)
	log.SetFormatter(&log.TextFormatter{})
    log.SetLevel(log.WarnLevel)
	
	// Map arguments
	globals.Arguments = make(map[string]interface{})
	globals.Arguments["--debug"] = false
	
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(windowWidth, windowHeight, "CMD")
	rl.SetWindowMinSize(windowWidth, windowHeight)

	rl.SetWindowIcon(*rl.LoadImage(ASSET_PATH + "images/icon.png"))
	
	icon := *rl.LoadImage(ASSET_PATH + "images/nav.png")
	rl.SetWindowIcon(icon)

	rl.SetTargetFPS(60)

	rl.SetExitKey(0) // Disable ESC key behavior

	loadScreenDuration := float32(0)
	loadScreen := rl.LoadTexture(ASSET_PATH + "images/load.png")
	bg := rl.LoadTexture(ASSET_PATH + "images/background_1.png")
	tg := rl.LoadTexture(ASSET_PATH + "images/background_x.png")
	loadScreenColor := rl.White
	
	logo := rl.LoadTexture(ASSET_PATH + "images/nav.png")
	dero := rl.LoadTexture(ASSET_PATH + "images/dero.png")

	font := *rl.LoadFontEx(FONT_PATH + "ebrima.ttf", int(fontSize), nil, 256)
	fontMenu := *rl.LoadFontEx(FONT_PATH + "Raleway.ttf", int(40), nil, 256)
	fontHeader := *rl.LoadFontEx(FONT_PATH + "Raleway.ttf", int(30), nil, 256)
	fontSubHeader := *rl.LoadFontEx(FONT_PATH + "Raleway.ttf", int(25), nil, 256)
	fontBalance := *rl.LoadFontEx(FONT_PATH + "Electrolize.ttf", int(55), nil, 256)
	fontPending := *rl.LoadFontEx(FONT_PATH + "Electrolize.ttf", int(20), nil, 256)
	fontPassword := *rl.LoadFontEx(FONT_PATH + "Raleway.ttf", int(50), nil, 256)
	
	cmdGray = rl.NewColor(9, 17, 24, 255)
	cmdBlue = rl.NewColor(0, 210, 255, 255)
	cmdGreen = rl.NewColor(22, 238, 88, 255)
	transparent = rl.NewColor(0,0,0,0)
	
	rl.GuiSetFont(fontSubHeader)
	
	session.Color = cmdGreen
	session.Network = "Mainnet"
	
	// Check/load config 
	if _, err = os.Stat(APP_PATH + "config.json"); err != nil {
		data := Config{
			RemoteDaemon:			DEFAULT_REMOTE_NODE,
			RemoteDaemonTestnet:	DEFAULT_REMOTE_NODE_TESTNET,
			LocalDaemon:			DEFAULT_LOCAL_NODE,
			LocalDaemonTestnet:		DEFAULT_LOCAL_NODE_TESTNET,
			RPCAuth:				"",
			RPCAddress:				"127.0.0.1:20209",
			DefaultMode:			"remote",
			DefaultNetwork:			"Mainnet",
		}
		
		file, _ := json.MarshalIndent(data, "", " ")
		_ = ioutil.WriteFile("config.json", file, 0644)
		
		session.Mode = "remote"
		session.Daemon = DEFAULT_REMOTE_NODE
		session.Rescan = false
		session.Syncing = false
		session.Network = "Mainnet"
		session.RPCServer = false
		session.RPCAddress = "127.0.0.1:20209"
		
		config.RemoteDaemon = DEFAULT_REMOTE_NODE
		config.RemoteDaemonTestnet = DEFAULT_REMOTE_NODE_TESTNET
		config.LocalDaemon = DEFAULT_LOCAL_NODE
		config.LocalDaemonTestnet = DEFAULT_LOCAL_NODE_TESTNET
		config.DefaultMode = "remote"
		config.DefaultNetwork = "Mainnet"
		config.RPCAddress = "127.0.0.1:20209"
		
		globals.Arguments["--testnet"] = false
		log.Warnf("File not found, config.json created.")
	} else {
		file, err := os.Open(APP_PATH + "config.json")
		
		if err == nil {
			defer file.Close()
			
			byteValue, _ := ioutil.ReadAll(file)
			json.Unmarshal(byteValue, &config)
			
			session.Network = config.DefaultNetwork
			
			if session.Network == "Mainnet" {
				if config.DefaultMode == "remote" {
					session.Daemon = config.RemoteDaemon
					session.Mode = "remote"
				} else if config.DefaultMode == "local" {
					session.Daemon = config.LocalDaemon
					session.Mode = "local"
				} else if config.DefaultMode == "" {
					session.Daemon = DEFAULT_REMOTE_NODE
					session.Mode = "remote"
				} else {
					session.Daemon = ""
					session.Mode = "offline"
				}
				
				session.RPCAddress = "127.0.0.1:20209"
				globals.Arguments["--testnet"] = false
			} else {
				if config.DefaultMode == "remote" {
					session.Daemon = config.RemoteDaemonTestnet
					session.Mode = "remote"
				} else if config.DefaultMode == "local" {
					session.Daemon = config.LocalDaemonTestnet
					session.Mode = "local"
				} else if config.DefaultMode == "" {
					session.Daemon = DEFAULT_REMOTE_NODE_TESTNET
					session.Mode = "remote"
				} else {
					session.Daemon = ""
					session.Mode = "offline"
				}
				
				session.RPCAddress = "127.0.0.1:30309"
				globals.Arguments["--testnet"] = true
			}
			
			session.Rescan = false
			session.Syncing = false
			session.RPCServer = false
		} else {
			session.Network = "Mainnet"
			session.Mode = "remote"
			session.Daemon = DEFAULT_REMOTE_NODE
			session.Rescan = false
			session.Syncing = false
			session.Network = "Mainnet"
			session.RPCServer = false
			session.RPCAddress = "127.0.0.1:20209"
			globals.Arguments["--testnet"] = false
			
			log.Warnf("Error loading config.json, loading default settings.")
		}
	}
	
	if session.Network == "Mainnet" {
		session.Color = cmdGreen
	} else {
		session.Color = cmdBlue
	}
	
	globals.Initialize()
	
	for !rl.WindowShouldClose() {
	
		reloadConfig()
		
		rl.GuiSetStyle(0, 1, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(0, 2, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(0, 3, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(0, 4, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(0, 5, rl.ColorToInt(session.Color)) // label fore color
		rl.GuiSetStyle(0, 6, rl.ColorToInt(session.Color)) // button text border
		rl.GuiSetStyle(0, 7, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(0, 8, rl.ColorToInt(session.Color)) // textbox active background
		rl.GuiSetStyle(0, 9, rl.ColorToInt(cmdGray))
		
		rl.GuiSetStyle(1, 1, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 2, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 3, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 4, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(1, 5, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(1, 6, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 7, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 8, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(1, 9, rl.ColorToInt(cmdGray))
		
		rl.GuiSetStyle(2, 1, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(2, 2, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 3, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 4, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(2, 5, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 6, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 7, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 8, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(2, 9, rl.ColorToInt(session.Color))
		
		rl.GuiSetStyle(3, 1, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(3, 2, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 3, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 4, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 5, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 6, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 7, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 8, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(3, 9, rl.ColorToInt(session.Color))

		rl.GuiSetStyle(rl.GuiControlLabel, rl.GuiPropertyTextColorNormal, rl.ColorToInt(rl.White))
		rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyTextColorNormal, rl.ColorToInt(rl.White))
		rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyBaseColorNormal, rl.ColorToInt(cmdGray))
		rl.GuiSetStyle(rl.GuiControlDropDownBox, rl.GuiPropertyTextColorNormal, rl.ColorToInt(session.Color))
		rl.GuiSetStyle(rl.GuiControlDropDownBox, rl.GuiPropertyBaseColorNormal, rl.ColorToInt(cmdGray))
	
		if rl.IsKeyPressed(rl.KeyEscape) {
			if (isFullscreen == true) {
				rl.SetWindowSize(windowWidth, windowHeight)
				isFullscreen = false
				statusBar = rl.Rectangle{0, float32(rl.GetScreenWidth()) - 32, float32(rl.GetScreenHeight()), 32}
				statusBarY = float32(rl.GetScreenHeight()) - 30
				statusBarWidth = float32(rl.GetScreenWidth())
				rl.ToggleFullscreen()
				rl.SetWindowPosition(60, 40)
			}
		}

		if rl.IsKeyPressed(rl.KeyF1) {
			if (isFullscreen == false) {
				rl.SetWindowSize(rl.GetMonitorWidth(0), rl.GetMonitorHeight(0))
				isFullscreen = true
				rl.ToggleFullscreen()
				statusBar = rl.Rectangle{0, float32(rl.GetMonitorHeight(0)) - 32, float32(rl.GetMonitorWidth(0)), 32}
			} else {
				rl.SetWindowSize(windowWidth, windowHeight)
				isFullscreen = false
				statusBar = rl.Rectangle{0, float32(rl.GetScreenWidth()) - 32, float32(rl.GetScreenHeight()), 32}
				statusBarY = float32(rl.GetScreenHeight()) - 30
				statusBarWidth = float32(rl.GetScreenWidth())
				rl.ToggleFullscreen()
				rl.SetWindowPosition(60, 40)
			}
		}

		rl.ClearBackground(rl.Black)
		rl.BeginDrawing()
		
		color := rl.White
		color.A = 200
		color = rl.White
		statusBG := rl.Black
		statusBG.A = 60
		rect80 := rl.Black
		rect80.A = 95
		terminalBG := rl.Black
		terminalBG.A = 250

		loadScreenDuration += rl.GetFrameTime()

		if loadScreenDuration >= 3.5 {
			if loadScreenColor.A > 5 {
				loadScreenColor.A -= 5
			} else {
				loadScreenColor.A = 0
			}
		}
		
		statusBar = rl.Rectangle{0, float32(rl.GetScreenHeight()) - 32, float32(rl.GetScreenWidth()), 32}
		progressBar = rl.Rectangle{0, float32(rl.GetScreenHeight()) - 36, float32(progressNow), 4}
	
		currentTime := time.Now().Format(time.UnixDate)
		currentTimeWidth = rl.MeasureTextEx(font, currentTime, fontSize, spacing)

		src = rl.Rectangle{0, 0, float32(loadScreen.Width), float32(loadScreen.Height)}
		dst = rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(loadScreen, src, dst, rl.Vector2{0,0}, 0, loadScreenColor)
		
		statusBarY = float32(rl.GetScreenHeight()) - 30
		statusBarWidth = float32(rl.GetScreenWidth())
		offsetX := float32(300)
		
		if loadScreenColor.A == 0 {
			syncStatus()
			// Background
			src := rl.Rectangle{0, 0, float32(bg.Width), float32(bg.Height)}
			src2 := rl.Rectangle{0, 0, float32(tg.Width), float32(tg.Height)}
			dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight() + 30)}
			
			if windowIndex == 0 {
				rl.DrawTexturePro(tg, src2, dst, rl.Vector2{0,0}, 0, rl.White)
			} else {
				rl.DrawTexturePro(bg, src, dst, rl.Vector2{0,30}, 0, rl.White)
			}
			
			// Status Bar
			rl.DrawRectangleRec(statusBar, statusBG)
			rl.DrawRectangleRec(progressBar, session.Color)
			rl.DrawLine(int(statusBarX), int(statusBarY - 1), int(statusBarX + statusBarWidth), int(statusBarY - 1), rl.DarkGray)
			rl.DrawTextEx(font, "--" + session.Network + "-- " + statusText + "" + walletText, rl.Vector2{25, statusBarY + 3}, fontSize, spacing, statusColor)
			rl.DrawTextEx(font, currentTime, rl.Vector2{float32(rl.GetScreenWidth()) - currentTimeWidth.X - 25, statusBarY + 3}, fontSize, spacing, rl.White)
			// Logo
			rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyBorderColorNormal, 00000000)
			rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyBorderColorFocused, 00000000)

			rl.DrawTexture(logo, 30, 0, session.Color)

			rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyBorderColorNormal, rl.ColorToInt(rl.Gray))
			rl.GuiSetStyle(rl.GuiControlButton, rl.GuiPropertyBorderColorFocused, rl.ColorToInt(session.Color))
			
			if windowIndex < 2.2 {
				if rl.GuiLabelButton(rl.NewRectangle(30, 150, 40, 40), "Create") {
					if wallet == nil {
						windowIndex = 1.0
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 200, 40, 40), "View") {
					if wallet == nil {
						windowIndex = 2.0
					} else {
						windowIndex = 2.2
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 250, 40, 40), "Restore") {
					if wallet == nil {
						restoreFilename = ""
						restorePassword = ""
						restoreHex = ""
						fileError = false
						windowIndex = 3.0
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, float32(rl.GetScreenHeight() - 120), 40, 40), "Settings") {
					if wallet == nil {
						windowIndex = 4.0
					}
				}
			} else if windowIndex < 3.0 {
				if rl.GuiLabelButton(rl.NewRectangle(30, 150, 40, 40), "Overview") {
					if wallet != nil {
						windowIndex = 2.2
					}					
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 200, 40, 40), "Send") {
					if wallet != nil {
						windowIndex = 2.5
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 250, 40, 40), "Receive") {
					if wallet != nil {
						windowIndex = 2.4
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 300, 40, 40), "View History") {
					if wallet != nil {
						windowIndex = 2.6
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 350, 40, 40), "Rescan") {
					if wallet != nil {
						windowIndex = 2.3
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 400, 40, 40), "Options") {
					if wallet != nil {
						windowIndex = 2.7
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, float32(rl.GetScreenHeight() - 120), 40, 40), "Log Out") {
					closeWallet()
				}
			} else if (windowIndex >= 3.0 && windowIndex < 4.0) {
				if rl.GuiLabelButton(rl.NewRectangle(30, 150, 40, 40), "Create") {
					if wallet == nil {
						windowIndex = 1.0
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 200, 40, 40), "View") {
					if wallet == nil {
						windowIndex = 2.0
					} else {
						windowIndex = 2.2
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 250, 40, 40), "Restore") {
					if wallet == nil {
						restoreFilename = ""
						restorePassword = ""
						restoreHex = ""
						fileError = false
						windowIndex = 3.0
					}
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, float32(rl.GetScreenHeight() - 120), 40, 40), "Settings") {
					windowIndex = 4.0
				}			
			} else if (windowIndex >= 4.0 && windowIndex < 5.0) {
				if rl.GuiLabelButton(rl.NewRectangle(30, 150, 40, 40), "Network") {
					windowIndex = 4.1			
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 200, 40, 40), "Network Mode") {
					windowIndex = 4.2
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 250, 40, 40), "Daemon") {
					windowIndex = 4.3
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 300, 40, 40), "RPC Server") {
					windowIndex = 4.4
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, 350, 40, 40), "Version") {
					windowIndex = 4.5
				}
				if rl.GuiLabelButton(rl.NewRectangle(30, float32(rl.GetScreenHeight() - 120), 40, 40), "Back") {
					windowIndex = 0
				}
			}
				
			switch windowIndex {
			case 0:
				clearVars()
				statusColor = rl.White
				statusText = "Ready"

				break
				
			// Create an account
			case 1.0:
				if fileError == true {
					rl.DrawTextEx(font, "That account name already exists.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
				}

				createAccountCompleted = false
				statusColor = rl.White
				statusText = "Ready"
				seed = ""
				address_s = ""
				passwordError = false
				createAccountFilenameEditable = true
				
				rl.DrawTextEx(fontMenu, "Create an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Account Name /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Name:  (Example: Expenses)", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, createAccountFilename = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), createAccountFilename, 60, createAccountFilenameEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				createAccountFilename = strings.Trim(createAccountFilename, ".")
				createAccountFilename = strings.Trim(createAccountFilename, " ")
				masked, mask := textMask(createAccountFilename)
				if masked {
					rl.DrawTextEx(fontHeader, mask, rl.Vector2{offsetX, 231}, 30, spacing, session.Color)
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if createAccountFilename == "" {
						break
					}
					
					if _, err = os.Stat(APP_PATH + createAccountFilename + ".db"); err != nil {
						fileError = false
						windowIndex = 1.1
					} else {
						createAccountFilename = ""
						fileError = true
					}
				}
				break
				
			case 1.1:
				if passwordError == true {
					rl.DrawTextEx(font, "The provided passwords do not match.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
				}
				createAccountPasswordEditable = true
				rl.DrawTextEx(fontMenu, "Create an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Account Name / Password /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, createAccountPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), createAccountPassword, 160, createAccountPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(createAccountPassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if createAccountPassword != "" {
						windowIndex = 1.12
					}
				}
				break
				
			case 1.12:
				tempPasswordEditable = true
				rl.DrawTextEx(fontMenu, "Create an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Account Name / Password / Confirm Password /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Confirm Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, tempPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), tempPassword, 160, tempPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(tempPassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Submit") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if (len(createAccountPassword) < 1) {
						
					} else {
						if tempPassword == createAccountPassword {
							windowIndex = 1.2
						} else {
							passwordError = true
							createAccountPassword = ""
							tempPassword = ""
							windowIndex = 1.1
						}
					}
				}
				break
				
			case 1.2:
				if createAccountCompleted == false {
					wallet, err := walletapi.Create_Encrypted_Wallet_Random(createAccountFilename + ".db", createAccountPassword)
					if err != nil {
						wallet = nil
						windowIndex = 1.0
						log.Warnf("Error creating wallet: %s", err)
						// TODO: Add error-feedback for the user.
					} else {
						err = wallet.Set_Encrypted_Wallet_Password(createAccountPassword)
						if err != nil {
							log.Warnf("Error changing password: %s", err)
						}
						// TODO wallet.Set_Seed_Language(language)
						session.Path = createAccountFilename + ".db"
						currentAccountPassword = createAccountPassword
						createAccountCompleted = true
						createAccountFilename = ""
						createAccountPassword = ""
						tempPassword = ""
						passwordError = false
						seed = wallet.GetSeed()
						address_s = wallet.GetAddress().String()
							
						wallet.Close_Encrypted_Wallet()
					}
				} else {
					rl.DrawTextEx(fontMenu, "Account Status", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
					rl.DrawTextEx(font, "Please review the following account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
					rl.DrawTextEx(fontHeader, "Your account address:", rl.Vector2{offsetX, 150}, 30, spacing, rl.White)
					rl.DrawTextEx(font, address_s, rl.Vector2{offsetX, 200}, 20, spacing, session.Color)
					rl.DrawRectangleRec(rl.NewRectangle(offsetX, 300, 840, 150), rect80)
					rl.DrawTextRec(fontSubHeader, seed, rl.NewRectangle(offsetX + 20, 320, 800, 150), 25, spacing, true, rl.White)
					rl.DrawRectangleRec(rl.NewRectangle(offsetX, 470, 840, 35), session.Color)
					rl.DrawTextEx(fontSubHeader, "Keep these recovery words safe, or you may lose your account forever.", rl.Vector2{offsetX + 10, 475}, 25, spacing, rl.Black)
							
					if rl.GuiButton(rl.NewRectangle(offsetX, float32(rl.GetScreenHeight() - 130), 250, 50), "Copy Seed") {
						rl.SetClipboardText(seed)
					}
						
					if rl.GuiButton(rl.NewRectangle(590, float32(rl.GetScreenHeight() - 130), 250, 50), "View Account") {
						windowIndex = 2.1
					}
				}
				break
			
			// View an account
			case 2.0:
				clearVars()
				statusColor = rl.White
				statusText = "Ready"
				passwordError = false
					
				if (wallet != nil && session.Path != "") {
					windowIndex = 2.2
				}
				
				fileCount := 0
				
				/*				
				matches, _ := filepath.Glob(APP_PATH + "*")
				for _, match := range matches {
					check, _ := os.Stat(match)
					if !check.IsDir() {
						if strings.Contains(match, ".") {
						} else if match == "CMD" {
						} else {
							if fileCount > 0 {
								files += ";" + match
								fileCount += 1
							} else {
								files = match
								fileCount += 1
							}
						}
					}
				}
				*/
				
				matches, _ := filepath.Glob(APP_PATH + "*.db")
				for _, match := range matches {
					check, _ := os.Stat(match)
					if !check.IsDir() {
						if strings.Contains(match, ".db") && !strings.Contains(match, ".lock") {
							if fileCount > 0 {
								files += ";" + match
								fileCount += 1
							} else {
								files = match
								fileCount += 1
							}
						}
					}
				}
				
				fileList := strings.Split(files, ";")

				if (fileCount < 1) {
					rl.DrawTextEx(fontHeader, "Oops, Looks like there are no accounts here.", rl.Vector2{offsetX, 150}, 30, spacing, rl.Orange)
				} else {
					loginPassword = ""
					rl.DrawTextEx(fontMenu, "Select an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
					rl.DrawTextEx(font, "Choose from the previously created accounts below.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)

					if (rl.GuiButton(rl.NewRectangle(offsetX, 250, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
						if (session.Path != "") {
							windowIndex = 2.1
						}
					}
					
					combo := rl.GuiComboBox(rl.NewRectangle(offsetX, 150, 600, 50), files, active)
					
					if combo != active {
						active = combo
					}
					
					session.Path = fileList[active]
				}
				break
				
			case 2.1:
				if passwordError == true {
					rl.DrawTextEx(font, "The password entered is incorrect.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
				}
				rl.DrawTextEx(fontMenu, "Sign-in to an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Enter your password below to gain access to your account.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontHeader, session.Path, rl.Vector2{offsetX, 150}, 30, spacing, rl.White)
				_, loginPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), loginPassword, 60, loginPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(loginPassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Sign In") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if loginPassword != "" {
						windowIndex = 2.2
					}
				}
				break
				
			case 2.2:
				if session.Path == "" {
					windowIndex = 2.0
				} else {
					if wallet == nil {
						wallet, err = walletapi.Open_Encrypted_Wallet(session.Path, loginPassword)
							
						if err != nil {
							loginPassword = ""
							passwordError = true
							windowIndex = 2.1
							log.Warnf("Error opening wallet: %s", err)
							break
						}
					}
					
					loginPassword = ""
					passwordError = false
					resetTransfer()
					
					wallet.SetDaemonAddress(session.Daemon)
						
					if session.Mode != "offline" {
						if (session.Rescan && session.Path == rescanPath) {
							session.Rescan = false
							rescan_bc(wallet, 0)
							session.Syncing = true
						} else {
							wallet.SetOnlineMode()
							session.Syncing = true
						}
					} else {
						wallet.SetOfflineMode()
						session.Syncing = false
					}
						
					var viewOnly string
						
					if wallet.Is_View_Only() {
						viewOnly = "(View-Only)"
					} else {
						viewOnly = ""
					}
					
					balance, pending := wallet.Get_Balance()
						
					rl.DrawTextEx(fontMenu, "Account Overview", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
					rl.DrawTextEx(font, "Welcome Commander, here you can view the status of your active account, manage application settings, and more.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
					rl.DrawTexture(dero, int(offsetX), 250, rl.White)
					rl.DrawTextEx(fontBalance, globals.FormatMoney8(balance) + pendingText, rl.Vector2{offsetX + 65, 250}, 55, spacing, rl.White)
					rl.DrawTextEx(font, "PENDING", rl.Vector2{offsetX + 68, 304}, 20, spacing, rl.White)
					rl.DrawTextEx(fontPending, globals.FormatMoney8(pending), rl.Vector2{offsetX + 140, 305}, 20, spacing, rl.White)

					rl.DrawTextEx(fontHeader, session.Path + "  " + viewOnly , rl.Vector2{offsetX, 150}, 30, spacing, session.Color)
					rl.DrawTextEx(fontSubHeader, trim_address(wallet.GetAddress().String()), rl.Vector2{offsetX, 200}, 25, spacing, rl.White)
					
					available := true
					in := true
					out := true
					pool := true
					failed := false
					min_height := uint64(0)
					max_height := uint64(0)
					
					transfers := wallet.Show_Transfers(available, in, out, pool, failed, false, min_height, max_height)
					
					if len(transfers) == 0 {
						if (rl.GuiLabelButton(rl.NewRectangle(offsetX, 350, 200, 50), "Last Transaction")) {
							
						}
						rl.DrawTextEx(font, "No transaction history.", rl.Vector2{offsetX, 410}, 20, spacing, rl.Gray)
					} else {
						var record Record
						record.txNumber = 0
						record.txTime = string(transfers[record.txNumber].Time.Format(time.RFC822))
						record.txHeight = strconv.FormatUint(transfers[record.txNumber].Height, 10)
						record.txTopoHeight = strconv.FormatInt(transfers[record.txNumber].TopoHeight, 10)
						record.txID = transfers[record.txNumber].TXID.String()
						record.txPaymentID = string(transfers[record.txNumber].PaymentID)
						record.txAmount = globals.FormatMoney12(transfers[record.txNumber].Amount)
						//record.txKey = string(wallet.GetTXKey(crypto.HexToHash(record.txID)))
						record.txColor = rl.Gray
						
						switch transfers[record.txNumber].Status {
						case 0:
							record.txColor = session.Color
							rl.DrawTextEx(font, record.txTime + "\nTransaction ID:  " + record.txID + "\nRECEIVED  " + record.txAmount + "\n[ " + record.txHeight + " / " + record.txTopoHeight + " ]", rl.Vector2{offsetX + 30, 410}, 20, spacing, rl.White)
							break
						case 1:
							record.txColor = rl.Magenta
							rl.DrawTextEx(font, record.txTime + "\nTransaction ID:  " + record.txID + "\nSPENT  " + record.txAmount + "\n[ " + record.txHeight + " / " + record.txTopoHeight + " ]", rl.Vector2{offsetX + 30, 410}, 20, spacing, rl.White)
							break
						case 2:
							fallthrough
						default:
							record.txColor = rl.Gray
							rl.DrawTextEx(font, record.txTime + "\nTransaction ID:  " + record.txID + "\nTransaction status unknown\n" + string(transfers[0].Status), rl.Vector2{offsetX + 30, 410}, 20, spacing, rl.White)
						}
						
						rl.DrawLineEx(rl.Vector2{offsetX + 5, 410}, rl.Vector2{offsetX + 5, 525}, 4.0, record.txColor)
						
						if (rl.GuiLabelButton(rl.NewRectangle(offsetX, 350, 200, 50), "Last Transaction")) {
							openURL("http://explorer.dero.io/tx/" + record.txID)
						}
					}
					
					rl.DrawLineEx(rl.Vector2{1100, 150}, rl.Vector2{1100, 525}, 1.0, rl.DarkGray)
					
					if rl.GuiLabelButton(rl.NewRectangle(1150, 145, 200, 50), "RPC Server") {
						if session.RPCServer == false {
							if session.Mode != "offline" {
								if session.Network == "Testnet" {
									err = wallet.Start_RPC_Server(DEFAULT_RPC_ADDRESS_TESTNET)
									session.RPCAddress = DEFAULT_RPC_ADDRESS_TESTNET
								} else {
									err = wallet.Start_RPC_Server(DEFAULT_RPC_ADDRESS)
									session.RPCAddress = DEFAULT_RPC_ADDRESS
								}
								
								if err == nil {
									session.RPCServer = true
								} else {
									log.Warnf("Error starting RPC server: %s", err)
								}
							}
						} else {
							session.RPCServer = false
							wallet.Stop_RPC_Server()
						}
					}
					
					if session.RPCServer == true {
						rl.DrawTextEx(font, "ONLINE @ " + session.RPCAddress, rl.Vector2{1150, 195}, 20, 0, session.Color)
					} else {
						rl.DrawTextEx(font, "OFFLINE", rl.Vector2{1150, 195}, 20, 0, rl.Gray)
					}
					
					modeText := ""
					modeColor := rl.Gray
					
					if session.Mode == "offline" {
						modeText = "OFFLINE"
						modeColor = rl.Gray
					} else {
						modeText = strings.ToUpper(session.Mode) + " @ " + session.Daemon
						modeColor = session.Color
					}
					
					if rl.GuiLabelButton(rl.NewRectangle(1150, 240, 200, 50), "Network Mode") {
						toggleMode()
					}
					
					rl.DrawTextEx(font, modeText, rl.Vector2{1150, 295}, 20, 0, modeColor)
					rl.DrawTextEx(fontSubHeader, "My Node", rl.Vector2{1150, 355}, 25, 0, rl.White)
					
					if session.Mode == "local" {
						if dHeight == 0 {
							rl.DrawTextEx(font, "N/A", rl.Vector2{1150, 395}, 20, 0, rl.Gray)
						} else {
							rl.DrawTextEx(font, "Daemon TopoHeight:  " + strconv.FormatUint(dHeight, 10), rl.Vector2{1150, 395}, 20, 0, session.Color)
						}
					} else {
						rl.DrawTextEx(font, "N/A", rl.Vector2{1150, 395}, 20, 0, rl.Gray)
					}
					
					rl.DrawTextEx(fontSubHeader, "Sync Status", rl.Vector2{1150, 455}, 25, 0, rl.White)
					
					status := "Checking..."
					percent := float64(0)
					statusColor := rl.Gray
					diff := int64(0)
					
					if (wHeight != 0 && dHeight != 0) {
						if int64(wHeight) <= int64(dHeight) {
							diff = int64(dHeight) - int64(wHeight)
							percent = (float64(wHeight) / float64(dHeight)) * 100
							percentS := strconv.FormatFloat(percent, 'f', 1, 64)
						
							if (diff > 5 && wHeight != 0) {
								status = "Syncing...  " + percentS + "%"
								statusColor = rl.Gray
							} else if diff < 5 {
								status = "Syncing Complete."
								statusColor = session.Color
							} else {
								status = "Syncing..."
								statusColor = rl.Gray
							}
						} else {
							status = "Checking..."
						}
					}
					
					if session.Mode == "offline" {
						status = "N/A"
					}
					
					rl.DrawTextEx(font, status, rl.Vector2{1150, 495}, 20, 0, statusColor)
					
					// Toggle Offline Mode
					if session.Mode == "offline" {
						if rl.GuiButton(rl.NewRectangle(float32(rl.GetScreenWidth() - 150), 30, 120, 50), "Offline") {
							session.Mode = "remote"
							session.Daemon = config.RemoteDaemon
							wallet.SetOnlineMode()
							session.Syncing = true
						}
					} else {
						if rl.GuiButton(rl.NewRectangle(float32(rl.GetScreenWidth() - 150), 30, 120, 50), "Online") {
							session.Mode = "offline"
							session.RPCServer = false
							wallet.Stop_RPC_Server()
							wallet.SetOfflineMode()
							session.Syncing = false
							wallet.Close_Encrypted_Wallet()
							wallet = nil
							windowIndex = 2.1
						}
					}
					
				}
				break
				
			// Rescan options	
			case 2.3:
				if wallet == nil {
					windowIndex = 2.0
					break
				}
				
				rl.DrawTextEx(fontMenu, "Rescan Settings", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Rescan the entire blockchain. This process can take longer than an hour to complete.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 150, 200, 50), "Start Rescan") {
					session.Rescan = true
					rescanPath = session.Path
					rescanHeight = stoUint64(rescanString)
					wallet.SetOfflineMode()
					session.Syncing = false
					wallet.Close_Encrypted_Wallet()
					wallet = nil
					windowIndex = 2.1
				}
				
				break
				
			// Receive DERO: Account Information	
			case 2.4:
				if wallet == nil {
					windowIndex = 2.0
					break
				}
				
				rl.DrawTextEx(fontMenu, "Receive Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Easily receive payments using your public account information below.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Public Account Address", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				rl.DrawTextEx(font, wallet.GetAddress().String(), rl.Vector2{offsetX, 200}, 20, spacing, rl.White)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 250, 210, 50), "Copy Address") {
					rl.SetClipboardText(wallet.GetAddress().String())
				}
				
				break
			
			// Send DERO	
			case 2.5:
				if wallet.Is_View_Only() {
					windowIndex = 2.2
					break
				}
				
				transfer.OfflineTX = false
				
				rl.DrawTextEx(fontMenu, "Send Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Send / Destination Address /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Destination Address:  (Ctrl-V to paste)", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				//_, receiverString = rl.GuiTextBox(rl.NewRectangle(offsetX, 200, 200, 50), receiverString, 198, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 910, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 910, 275}, 2.0, session.Color)
				
				masked, mask := textMask(receiverString)
				
				if masked {
					rl.DrawTextEx(font, mask, rl.Vector2{offsetX, 231}, 20, spacing, session.Color)
				}
				
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					receiverString = rl.GetClipboardText()
					
					if len(receiverString) < 128 {
						masked, mask = textMask(receiverString)
					}
				}
				
				pid := false
				ok := false
				
				// Validate Address
				transfer.rAddress, err = globals.ParseValidateAddress(receiverString)
				
				if err != nil {
					rl.DrawTextEx(font, "Invalid address format.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
					ok = false
					
					if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Clear") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
						receiverString = ""
					}
						
					break
				} else {
					rl.DrawTextEx(font, "Valid Address", rl.Vector2{offsetX, 285}, 20, spacing, session.Color)
					ok = true
				}
					
				a := *transfer.rAddress
				
				if a.IsIntegratedAddress() {
					//transfer.PaymentID, err = hex.DecodeString(pidString)
					
					if err == nil {
						pid = true
					}
				} else {
					pid = false
				}

				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if ok == true {
						if pid {
							windowIndex = 2.52
						} else {
							windowIndex = 2.51
						}
					}
				}
				
				break
			
			case 2.51:
				if wallet.Is_View_Only() {
					windowIndex = 2.2
					break
				}
				
				rl.DrawTextEx(fontMenu, "Send Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Send / Destination Address / Payment ID /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Payment ID (optional):", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				//_, pidString = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 200, 50), pidString, 128, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 900, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 900, 275}, 2.0, session.Color)
				
				masked, mask := textMask(pidString)
				
				if masked {
					rl.DrawTextEx(font, mask, rl.Vector2{offsetX, 231}, 20, spacing, session.Color)
				}
				
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					pidString = rl.GetClipboardText()
					masked, mask = textMask(pidString)
				}
				
				err = nil

				if (rl.GuiButton(rl.NewRectangle(offsetX + 230, 350, 210, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					windowIndex = 2.52
				}
				
				//if pidString != "" {
					if rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Clear") {
						pidString = ""
						transfer.PaymentID = nil
						break
					}
					
					//transfer.PaymentID, err = hex.DecodeString(pidString)
					
					if (len(pidString) == 8 || len(pidString) == 64) {
						transfer.PaymentID, err = hex.DecodeString(pidString)

						if err != nil {
							rl.DrawTextEx(font, "Invalid payment ID format.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
							break
						}
					} else {
						rl.DrawTextEx(font, "Invalid payment ID format.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
						break
					}
				//}
		
				break
			
			case 2.52:
				if wallet.Is_View_Only() {
					windowIndex = 2.2
					break
				}
				
				rl.DrawTextEx(fontMenu, "Send Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Send / Destination Address / Payment ID / Amount /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Amount:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, amountString = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 200, 50), amountString, 60, true)
				checked = rl.GuiCheckBox(rl.NewRectangle(offsetX + 520, 220, 20, 20), "  All", checked)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := textMask(amountString)
				
				if masked {
					rl.DrawTextEx(fontHeader, mask, rl.Vector2{offsetX, 231}, 30, spacing, session.Color)
				}
				
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					amountString = rl.GetClipboardText()
					masked, mask = textMask(amountString)
				}
				
				curBalance, _ := wallet.Get_Balance()
				
				if checked {
					amountString = globals.FormatMoney12(curBalance)
				}
				
				// Validate Amount
				ok := false
				transfer.Amount, err = globals.ParseAmount(amountString)
				
				if err != nil {
					ok = false
					rl.DrawTextEx(font, "Invalid amount format.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
					break
				} else {
					curBalance, _ := wallet.Get_Balance()
					if transfer.Amount > curBalance {
						ok = false
						rl.DrawTextEx(font, "Insufficient balance.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
						break
					} else {
						ok = true
					}
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) && ok == true) {

					addr_list := []address.Address{*transfer.rAddress}
					amount_list := []uint64{transfer.Amount}
					fees_per_kb := uint64(0)                    // fees  must be calculated by walletapi
					
					if checked {
						tx, inputs, input_sum, err := wallet.Transfer_Everything(*transfer.rAddress, hex.EncodeToString(transfer.PaymentID), 0, fees_per_kb, 5)
						_ = inputs
						
						if err != nil {
							// TODO: Error feedback
							log.Warnf("Error building the transaction: %s", err)
							break
						} else {
							amountString = globals.FormatMoney12(input_sum)
							transfer.InputSum = input_sum
							transfer.Fees = tx.RctSignature.Get_TX_Fee()
							transfer.Amount = curBalance - transfer.Fees
							transfer.Size = float32(len(tx.Serialize()))/1024.0
							transfer.OfflineTX = false
							transfer.TX = tx
							checked = false
							windowIndex = 2.53
						}
					} else {
						tx, inputs, input_sum, change, err := wallet.Transfer(addr_list, amount_list, 0, hex.EncodeToString(transfer.PaymentID), fees_per_kb, 0)
						_ = inputs
						
						if err != nil {
							log.Warnf("Error while building transaction: %s", err)
							break
						}
						
						if session.Mode == "offline" {
							transfer.OfflineTX = true
						} else {
							transfer.OfflineTX = false
						}
						
						transfer.Relay = build_relay_transaction(tx, inputs, input_sum, change, err, transfer.OfflineTX, amount_list)
					
						if transfer.Relay {
							checked = false
							windowIndex = 2.53
						} else {
							rl.DrawTextEx(font, "Error:  Unable to build the transfer.", rl.Vector2{offsetX, 515}, 20, spacing, rl.Magenta)
							break
						}
					}
				}
				
				break
				
			case 2.53:
				if wallet.Is_View_Only() {
					windowIndex = 2.2
					break
				}
				
				if passwordError {
					rl.DrawTextEx(font, "The password entered is incorrect.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Send Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "/ Send / Destination Address / Payment ID / Amount / Confirmation /", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "My Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				strAmount := globals.FormatMoney12(transfer.Amount)
				strFees := globals.FormatMoney12(transfer.Fees)
				rl.DrawTextEx(font, "TRANSACTION DETAILS\n\nDestination Address:  " + transfer.rAddress.String() + "\nPayment ID:  " + hex.EncodeToString(transfer.PaymentID) + "\nAmount:  " + strAmount + "\nFees:  " + strFees, rl.Vector2{offsetX, 350}, 20, spacing, rl.White)
				_, loginPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), loginPassword, 160, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(loginPassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 600, 210, 50), "Confirm") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					check := wallet.Check_Password(loginPassword)
					if check == false {
						loginPassword = ""
						passwordError = true
						break
					} else {
						if transfer.OfflineTX { // if its an offline tx, dump it to a file
							cur_dir, err := os.Getwd()
							if err != nil {
								break
							}
							
							transfer.TXID = transfer.TX.GetHash()
		
							filename := filepath.Join(cur_dir, transfer.TXID.String() + ".tx")
							err = ioutil.WriteFile(filename, []byte(hex.EncodeToString(transfer.TX.Serialize())), 0600)

							if err == nil {
								transfer.Filename = filename
								transfer.Status = "Success"
							} else {
								transfer.Status = "Failed"
								log.Warnf("Error building offline transaction: %s", err)
							}
						} else {

							loginPassword = ""
							passwordError = false
							err = wallet.SendTransaction(transfer.TX) // relay tx to daemon/network
				
							if err == nil {
								transfer.Status = "Success"
								transfer.TXID = transfer.TX.GetHash()
							} else {
								transfer.Status = "Failed"
								transfer.TXID = transfer.TX.GetHash()
								log.Warnf("Error relaying transaction: %s", err)
							}
						}
						
						windowIndex = 2.54
					}
				}
				
				if rl.GuiButton(rl.NewRectangle(offsetX + 230, 600, 210, 50), "Cancel") {
					resetTransfer()
					windowIndex = 2.2
				}
			
				break
				
			case 2.54:
				if wallet.Is_View_Only() {
					windowIndex = 2.2
					break
				}
				
				transferText := ""
				transferHeader := ""
				transferMessage := ""
				
				if transfer.Status == "Success" {
					if transfer.OfflineTX {
						transferText = "Your offline payment was saved."
						transferHeader = "Transaction Path:"
						transferMessage = APP_PATH + transfer.Filename
					} else {
						transferText = "Your payment has been sent successfully!"
						transferHeader = "Transaction ID:"
						transferMessage = transfer.TXID.String()
					}
				} else {
					transferText = "Your payment has been declined."
				}
				
				rl.DrawTextEx(fontMenu, "Send Payments", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Payment Status", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontHeader, transferText, rl.Vector2{offsetX, 150}, 30, spacing, session.Color)
				rl.DrawTextEx(fontSubHeader, transferHeader, rl.Vector2{offsetX, 200}, 25, spacing, rl.White)
				rl.DrawTextEx(font, transferMessage, rl.Vector2{offsetX, 250}, 20, spacing, rl.White)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Return") {
					resetTransfer()
					windowIndex = 2.2
				}
				
				if transfer.OfflineTX == false {
					if rl.GuiButton(rl.NewRectangle(offsetX + 230, 350, 250, 50), "View Transaction") {
						openURL("http://explorer.dero.io/tx/" + transfer.TXID.String())
					}
				}
				
				break
			
			// View History
			case 2.6:
				rl.DrawTextEx(fontMenu, "View History", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Access all information related to your transactions.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				//rl.DrawTextEx(fontHeader, "My Transactions", rl.Vector2{offsetX, 150}, 30, spacing, session.Color)
				
				record := make([]Record, 3)
		
				available := true
				in := true
				out := true
				pool := true
				failed := false
				min_height := uint64(0)
				max_height := uint64(0)
				
				transfers := wallet.Show_Transfers(available, in, out, pool, failed, false, min_height, max_height)
					
				if len(transfers) == 0 {
					rl.DrawTextEx(font, "No transaction history.", rl.Vector2{offsetX, 150}, 20, spacing, rl.Gray)
					break
				} else {
					increment := 3
					posY := float32(150)
					
					if len(transfers) < 3 {
						increment = len(transfers)
					}
					
					for i := 0; i < increment; i++ {
						if i == 0 {
							posY = 150
						} else if i == 1 {
							posY = 350
						} else {
							posY = 550
						}
						
						if i > len(transfers) {
							break
						} else if i < 0 {
							break
						} else if (start + i) >= len(transfers) {
							break
						}
						
						record[i].txNumber = uint64(start + i)
						record[i].txTime = string(transfers[record[i].txNumber].Time.Format(time.RFC822))
						record[i].txHeight = strconv.FormatUint(transfers[record[i].txNumber].Height, 10)
						record[i].txTopoHeight = strconv.FormatInt(transfers[record[i].txNumber].TopoHeight, 10)
						record[i].txID = transfers[record[i].txNumber].TXID.String()
						record[i].txPaymentID = string(transfers[record[i].txNumber].PaymentID)
						record[i].txAmount = globals.FormatMoney12(transfers[record[i].txNumber].Amount)
						record[i].txKey = string(wallet.GetTXKey(crypto.HexToHash(record[i].txID)))
						record[i].txColor = rl.Gray
						
						if record[i].txKey != "" && record[i].txKey != "None" {
							if (rl.GuiLabelButton(rl.NewRectangle(offsetX + 700, posY + 50, 200, 50), "Copy Transaction Key")) {
								rl.SetClipboardText(record[i].txKey)
							}
						} else {
							record[i].txKey = "None"
						}
						
						switch transfers[record[i].txNumber].Status {
						case 0:
							record[i].txColor = session.Color
							rl.DrawTextEx(font, record[i].txTime + "\nTransaction ID:  " + record[i].txID + "\nRECEIVED  " + record[i].txAmount + "\n[ " + record[i].txHeight + " / " + record[i].txTopoHeight + " ]\nTransaction Key:  " + record[i].txKey, rl.Vector2{offsetX + 30, posY}, 20, spacing, rl.White)
							break
						case 1:
							record[i].txColor = rl.Magenta
							rl.DrawTextEx(font, record[i].txTime + "\nTransaction ID:  " + record[i].txID + "\nSPENT  " + record[i].txAmount + "\n[ " + record[i].txHeight + " / " + record[i].txTopoHeight + " ]\nTransaction Key:  " + record[i].txKey, rl.Vector2{offsetX + 30, posY}, 20, spacing, rl.White)
							break
						case 2:
							fallthrough
						default:
							record[i].txColor = rl.Gray
							rl.DrawTextEx(font, record[i].txTime + "\nTransaction ID:  " + record[i].txID + "\nTransaction status unknown\n" + string(transfers[i].Status), rl.Vector2{offsetX + 30, posY}, 20, spacing, rl.White)
						}
						
						rl.DrawLineEx(rl.Vector2{offsetX + 5, posY}, rl.Vector2{offsetX + 5, posY + 145}, 4.0, record[i].txColor)
						rl.DrawLineEx(rl.Vector2{offsetX + 680, posY}, rl.Vector2{offsetX + 680, posY + 145}, 1.0, rl.DarkGray)
						
						if (rl.GuiLabelButton(rl.NewRectangle(offsetX + 700, posY, 200, 50), "View in Explorer")) {
							openURL("http://explorer.dero.io/tx/" + record[i].txID)
						}
						
						if (rl.GuiLabelButton(rl.NewRectangle(float32(rl.GetScreenWidth() - 300), 30, 100, 50), "< Previous")) {
							if start >= 3 {
								start -= 3
							}
						}
						
						if (rl.GuiLabelButton(rl.NewRectangle(float32(rl.GetScreenWidth() - 130), 30, 100, 50), "Next >")) {
							if start < len(transfers) - 2 {
								start += 3
							}
						}
					}
				}
					
				break
				
			// Options
			case 2.7:
				if passwordError == true {
					rl.DrawTextEx(font, "The password entered was incorrect.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Account Options", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update your account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "My Password", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, loginPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), loginPassword, 160, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(loginPassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Confirm") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					check := wallet.Check_Password(loginPassword)
					if check == false {
						loginPassword = ""
						passwordError = true
						break
					} else {
						passwordError = false
						loginPassword = ""
						windowIndex = 2.71
					}
				}
				
				break
				
			// Options Menu
			case 2.71:
				clearVars()
				rl.DrawTextEx(fontMenu, "Account Options", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update your account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				
				// Seed
				rl.DrawTextEx(fontSubHeader, "Your 25 recovery words (seed) can be used to restore your account.", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 210, 210, 50), "Copy Seed") {
					rl.SetClipboardText(wallet.GetSeed())
				}
				
				// View wallet key
				rl.DrawTextEx(fontSubHeader, "A view key can be used to restore a view-only account.", rl.Vector2{offsetX, 320}, 25, spacing, rl.White)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 380, 210, 50), "Copy Key") {
					rl.SetClipboardText(wallet.GetViewWalletKey())
				}
				
				// Change Password
				rl.DrawTextEx(fontSubHeader, "Update your account password.", rl.Vector2{offsetX, 490}, 25, spacing, rl.White)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 550, 210, 50), "Update") {
					windowIndex = 2.72
				}
				
				break
				
			// Change Password
			case 2.72:
				if passwordError == true {
					rl.DrawTextEx(font, "The passwords entered do not match.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Account Options", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update your account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "New Password", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, loginPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), loginPassword, 160, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(loginPassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					windowIndex = 2.73
				}
				
				break
				
			// Change Password cont.
			case 2.73:
				rl.DrawTextEx(fontMenu, "Account Options", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update your account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Confirm Password", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, tempPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), tempPassword, 160, true)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(tempPassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
				
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 210, 50), "Confirm") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if loginPassword == tempPassword {
						err = wallet.Set_Encrypted_Wallet_Password(loginPassword)
						
						if err != nil {
							// TODO: Error feedback
							log.Warnf("Error changing password: %s", err)
							break
						} else {
							passwordError = false
							loginPassword = ""
							tempPassword = ""
							windowIndex = 2.74
						}
					} else {
						loginPassword = ""
						tempPassword = ""
						passwordError = true
						windowIndex = 2.72
					}
				}
				
				break
				
			// Change Password cont.
			case 2.74:
				rl.DrawTextEx(fontMenu, "Account Options", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update your account information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Your password has been updated successfully.", rl.Vector2{offsetX, 150}, 25, spacing, session.Color)
				
				if rl.GuiButton(rl.NewRectangle(offsetX, 250, 210, 50), "Return") {
					windowIndex = 2.71
				}
				
				break
			
			// Restore an account
			case 3.0:
				clearVars()
				if wallet != nil {
					windowIndex = 2.2
				}
				
				restorePassword = ""
				tempPassword = ""

				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Choose one of the methods below to restore your account.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)

				if rl.GuiLabelButton(rl.NewRectangle(offsetX, 150, 250, 50), "_ View-Only using Hex (128 chars)") {
					windowIndex = 3.1
				}
				if rl.GuiLabelButton(rl.NewRectangle(offsetX, 250, 250, 50), "_ Restore using Seed (25 words)") {
					windowIndex = 3.2
				}
				if rl.GuiLabelButton(rl.NewRectangle(offsetX, 350, 250, 50), "_ Restore using Hex (64 chars)") {
					windowIndex = 3.3
				}
				break
			
			// Restore View-Only using Hex (128 chars)
			case 3.1:
				if fileError == true {
					rl.DrawTextEx(font, "That account name already exists.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore View-Only with Hex (128 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Name:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restoreFilename = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restoreFilename, 60, restoreFilenameEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				restoreFilename = strings.Trim(restoreFilename, ".")
				restoreFilename = strings.Trim(restoreFilename, " ")
				masked, mask := textMask(restoreFilename)
				if masked {
					rl.DrawTextEx(fontHeader, mask, rl.Vector2{offsetX, 231}, 30, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restoreFilename != "" {
						if _, err = os.Stat(APP_PATH + restoreFilename + ".db"); err != nil {
							fileError = false
							windowIndex = 3.12
						} else {
							fileError = true
						}
					}
				}
				break
				
			case 3.12:
				if passwordError == true {
					rl.DrawTextEx(font, "The passwords entered do not match.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore View-Only with Hex (128 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restorePassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restorePassword, 60, restorePasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(restorePassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" {
						windowIndex = 3.121
					}
				}
				break
				
			case 3.121:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore View-Only with Hex (128 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Confirm Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				tempPasswordUpdated, tempPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), tempPassword, 60, tempPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(tempPassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" && restorePassword != tempPassword {
						passwordError = true
						restorePassword = ""
						tempPassword = ""
						windowIndex = 3.12
					} else {
						passwordError = false
						windowIndex = 3.13
					}
				}
				break
				
			case 3.13:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore View-Only with Hex (128 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Enter your 128 character hex (Ctrl-V to Paste) :", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 300}, rl.Vector2{offsetX + 800, 300}, 2.0, session.Color)
					
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					restoreHex = string(rl.GetClipboardText())
				}
					
				rl.DrawTextRecEx(font, restoreHex, rl.NewRectangle(offsetX, 231, 800, 200), 20, 0, true, session.Color, 0, 0, rl.Gray, statusColor)
						
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if (restoreHex != "" && len(restoreHex) == 128) {
						windowIndex = 3.14
					}
				}
				break
				
			case 3.14:
				if (restoreFilename != "" && restorePassword != "" && restoreHex != "") {	
					wallet, err = walletapi.Create_Encrypted_Wallet_ViewOnly(restoreFilename + ".db", restorePassword, restoreHex)

					if err != nil {
						log.Warnf("Error while reconstructing view-only wallet using view key: %s", err)
					} else {
						err = wallet.Set_Encrypted_Wallet_Password(restorePassword)
						if err != nil {
							log.Warnf("Error changing password: %s", err)
						} else {
							closeWallet()
							windowIndex = 2.1
							session.Path = restoreFilename + ".db"
							restoreFilename = ""
							restorePassword = ""
							tempPassword = ""
							restoreHex = ""
							fileError = false
							passwordError = false
						}
					}
				}
				break
			
			// Restore using Seed (25 words)			
			case 3.2:
				if fileError == true {
					rl.DrawTextEx(font, "That account name already exists.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}

				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Seed (25 words)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Name:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restoreFilename = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restoreFilename, 60, restoreFilenameEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				restoreFilename = strings.Trim(restoreFilename, ".")
				restoreFilename = strings.Trim(restoreFilename, " ")
				masked, mask := textMask(restoreFilename)
				
				if masked {
					rl.DrawTextEx(fontHeader, mask, rl.Vector2{offsetX, 231}, 30, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restoreFilename != "" {
						if _, err = os.Stat(APP_PATH + restoreFilename + ".db"); err != nil {
							fileError = false
							windowIndex = 3.21
						} else {
							fileError = true
						}
					} else {
						// error handling
					}
				}
				break
				
			case 3.21:
				if passwordError == true {
					rl.DrawTextEx(font, "Passwords do not match.", rl.Vector2{offsetX, 285}, 20, spacing, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Seed (25 words)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restorePassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restorePassword, 160, restorePasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(restorePassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" {
						windowIndex = 3.22
					} else {
						// error handling
					}
				}
				break
				
			case 3.22:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Seed (25 words)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Confirm Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				tempPasswordUpdated, tempPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), tempPassword, 160, tempPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(tempPassword)
				
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" && restorePassword != tempPassword {
						passwordError = true
						restorePassword = ""
						tempPassword = ""
						windowIndex = 3.21
					} else {
						passwordError = false
						windowIndex = 3.23
					}
				}
				break
				
			case 3.23:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Seed (25 words)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Enter your 25 recovery words (Ctrl-V to Paste) :", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 300}, rl.Vector2{offsetX + 800, 300}, 2.0, session.Color)
					
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					restoreHex = string(rl.GetClipboardText())
				}
					
				rl.DrawTextRecEx(font, restoreHex, rl.NewRectangle(offsetX, 231, 800, 200), 20, 0, true, session.Color, 0, 0, rl.Gray, statusColor)
						
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restoreHex != "" {
						windowIndex = 3.24
					} else {
						// error handling
					}
				}
				break
				
			case 3.24:
				if (restoreFilename != "" && restorePassword != "" && restoreHex != "") {	
					wallet, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(restoreFilename + ".db", restorePassword, restoreHex)

					if err != nil {
						log.Warnf("Error while reconstructing view-only wallet using view key: %s", err)
					} else {
						err = wallet.Set_Encrypted_Wallet_Password(restorePassword)
						if err != nil {
							log.Warnf("Error changing password: %s", err)
						} else {
							closeWallet()
							session.Path = restoreFilename + ".db"
							windowIndex = 2.1
							restoreFilename = ""
							restorePassword = ""
							tempPassword = ""
							restoreHex = ""
							fileError = false
							passwordError = false
						}
					}
				}
				break
			
			// Restore using Hex (64 chars)
			case 3.3:
				if fileError == true {
					rl.DrawTextEx(font, "That account name already exists.", rl.Vector2{offsetX, 285}, 20, 0, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Hex (64 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Name:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restoreFilename = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restoreFilename, 60, restoreFilenameEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := textMask(restoreFilename)
				
				if masked {
					rl.DrawTextEx(fontHeader, mask, rl.Vector2{offsetX, 231}, 30, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restoreFilename != "" {
						if _, err = os.Stat(APP_PATH + restoreFilename + ".db"); err != nil {
							fileError = false
							windowIndex = 3.21
						} else {
							fileError = true
						}
					} else {
						// error handling
					}
				}
				break
				
			case 3.31:
				if passwordError == true {
					rl.DrawTextEx(font, "Passwords do not match.", rl.Vector2{400, 285}, 20, spacing, rl.Gray)
				}
				
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Hex (64 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Account Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, restorePassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), restorePassword, 160, restorePasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(restorePassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" {
						windowIndex = 3.22
					} else {
						// error handling
					}
				}
				break
				
			case 3.32:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Hex (64 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Confirm Password:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, tempPassword = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 400, 50), tempPassword, 160, tempPasswordEditable)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 500, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				
				masked, mask := passwordMask(tempPassword)
				if masked {
					rl.DrawTextEx(fontPassword, mask, rl.Vector2{offsetX, 231}, 50, spacing, session.Color)
				}
					
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restorePassword != "" && restorePassword != tempPassword {
						passwordError = true
						windowIndex = 3.21
					} else {
						passwordError = false
						windowIndex = 3.23
					}
				}
				break
				
			case 3.33:
				rl.DrawTextEx(fontMenu, "Restore an Account", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Restore using Hex (64 chars)", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Enter your 65 hex chars (Ctrl-V to Paste) :", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				//restoreHexUpdated, restoreHex = rl.GuiTextBoxMulti(rl.NewRectangle(offsetX, 300, 1000, 200), restoreHex, 64, restoreHexEditable)
					
				if (rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyDown(rl.KeyV)) {
					restoreHex = string(rl.GetClipboardText())
				}
					
				rl.DrawTextRecEx(font, restoreHex, rl.NewRectangle(offsetX, 231, 800, 200), 20, 0, true, session.Color, 0, 0, rl.Gray, statusColor)
						
				if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Next") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
					if restoreHex != "" {
						windowIndex = 3.24
					} else {
						// error handling
					}
				}
				break
				
			case 3.34:
				if (restoreFilename != "" && restorePassword != "" && restoreHex != "") {	
					wallet, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(restoreFilename + ".db", restorePassword, restoreHex)

					if err != nil {
						log.Warnf("Error while reconstructing view-only wallet using view key: %s", err)
					} else {
						err = wallet.Set_Encrypted_Wallet_Password(restorePassword)
						if err != nil {
							log.Warnf("Error changing password: %s", err)
						} else {
							closeWallet()
							session.Path = restoreFilename + ".db"
							windowIndex = 2.1
							restoreFilename = ""
							restorePassword = ""
							tempPassword = ""
							restoreHex = ""
							fileError = false
							passwordError = false
						}
					}
				}
				break
				
			case 4.0:
				clearVars()
				rl.DrawTextEx(fontMenu, "Settings", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "View and update application-specific settings.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				//rl.DrawTextEx(fontSubHeader, "Settings have been disabled in this release.", rl.Vector2{offsetX, 150}, 25, spacing, rl.Orange)
				break
			
			// Network
			case 4.1:
				clearVars()
				rl.DrawTextEx(fontMenu, "Network", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Choose your default network.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				
				active := 0
				
				if session.Network == "Mainnet" {
					active = 0
				} else {
					active = 1
				}
				
				networks := "Mainnet;Testnet"
				networkList := strings.Split(networks, ";")
				
				combo := rl.GuiComboBox(rl.NewRectangle(offsetX, 150, 600, 50), networks, active)
					
				if combo != active {
					active = combo
				}
					
				session.Network = networkList[active]
				config.DefaultNetwork = networkList[active]
	
				data := Config {
					RemoteDaemon:			config.RemoteDaemon,
					RemoteDaemonTestnet:	config.RemoteDaemonTestnet,
					LocalDaemon:			config.LocalDaemon,
					LocalDaemonTestnet:		config.LocalDaemonTestnet,
					RPCAuth:				config.RPCAuth,
					RPCAddress:				config.RPCAddress,
					DefaultMode:			config.DefaultMode,
					DefaultNetwork:			networkList[active],
				}
					
				file, _ := json.MarshalIndent(data, "", " ")
				_ = ioutil.WriteFile("config.json", file, 0644)
				
				break
				
			// Network Mode
			case 4.2:
				clearVars()
				rl.DrawTextEx(fontMenu, "Network Mode", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Choose your default network mode.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				
				active := 0
				
				if session.Mode == "remote" {
					active = 0
				} else if session.Mode == "local" {
					active = 1
				} else {
					active = 2
				}
				
				modes := "remote;local;offline"
				modeList := strings.Split(modes, ";")
				
				combo := rl.GuiComboBox(rl.NewRectangle(offsetX, 150, 600, 50), modes, active)
					
				if combo != active {
					active = combo
				}
					
				session.Mode = modeList[active]
				config.DefaultMode = modeList[active]
	
				data := Config {
					RemoteDaemon:			config.RemoteDaemon,
					RemoteDaemonTestnet:	config.RemoteDaemonTestnet,
					LocalDaemon:			config.LocalDaemon,
					LocalDaemonTestnet:		config.LocalDaemonTestnet,
					RPCAuth:				config.RPCAuth,
					RPCAddress:				config.RPCAddress,
					DefaultMode:			modeList[active],
					DefaultNetwork:			config.DefaultNetwork,
				}
					
				file, _ := json.MarshalIndent(data, "", " ")
				_ = ioutil.WriteFile("config.json", file, 0644)
				
				break
				
			// Daemon
			case 4.3:
				clearVars()
				em = true
				
				if config.RemoteDaemon != DEFAULT_REMOTE_NODE {
					checked = false
				}
				
				rl.DrawTextEx(fontMenu, "Daemon", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Configure remote daemon addresses.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(fontSubHeader, "Remote Node:", rl.Vector2{offsetX, 150}, 25, spacing, rl.White)
				_, config.RemoteDaemon = rl.GuiTextBox(rl.NewRectangle(offsetX, 220, 900, 50), config.RemoteDaemon, 260, em)
				rl.DrawRectangleRec(rl.NewRectangle(offsetX, 215, 950, 60), cmdGray)
				rl.DrawLineEx(rl.Vector2{offsetX, 275}, rl.Vector2{offsetX + 500, 275}, 2.0, session.Color)
				checked = rl.GuiCheckBox(rl.NewRectangle(offsetX + 520, 220, 20, 20), "  Default", checked)
				
				if em {
					masked, mask := textMask(config.RemoteDaemon)
					
					if masked {
						rl.DrawTextEx(font, mask, rl.Vector2{offsetX, 231}, 20, spacing, session.Color)
					}

					if (rl.GuiButton(rl.NewRectangle(offsetX, 350, 150, 50), "Save") || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {

					}
				}
				
				if checked {
					config.RemoteDaemon = DEFAULT_REMOTE_NODE
				}
	
				data := Config {
					RemoteDaemon:			config.RemoteDaemon,
					RemoteDaemonTestnet:	config.RemoteDaemonTestnet,
					LocalDaemon:			config.LocalDaemon,
					LocalDaemonTestnet:		config.LocalDaemonTestnet,
					RPCAuth:				config.RPCAuth,
					RPCAddress:				config.RPCAddress,
					DefaultMode:			config.DefaultMode,
					DefaultNetwork:			config.DefaultNetwork,
				}
					
				file, _ := json.MarshalIndent(data, "", " ")
				_ = ioutil.WriteFile("config.json", file, 0644)
				
				break
				
			// RPC Server
			case 4.4:
				clearVars()
				rl.DrawTextEx(fontMenu, "RPC Server", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Configure authentication and address for your RPC server.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				

	
				data := Config {
					RemoteDaemon:			config.RemoteDaemon,
					RemoteDaemonTestnet:	config.RemoteDaemonTestnet,
					LocalDaemon:			config.LocalDaemon,
					LocalDaemonTestnet:		config.LocalDaemonTestnet,
					RPCAuth:				config.RPCAuth,
					RPCAddress:				config.RPCAddress,
					DefaultMode:			config.DefaultMode,
					DefaultNetwork:			config.DefaultNetwork,
				}
					
				file, _ := json.MarshalIndent(data, "", " ")
				_ = ioutil.WriteFile("config.json", file, 0644)
				
				break
			
			case 4.5:
				clearVars()
				rl.DrawTextEx(fontMenu, "Version", rl.Vector2{offsetX, 30}, 40, spacing, rl.Gray)
				rl.DrawTextEx(font, "Application version information.", rl.Vector2{offsetX, 80}, 20, spacing, rl.Gray)
				rl.DrawTextEx(font, "CMD v" + Version.String() + "\n\nCopyright 2020 DERO Foundation. All rights reserved.\nPrivacy Together.", rl.Vector2{offsetX, 150}, 20, spacing, session.Color)
				break
			
			case 5.0:
				rl.DrawRectangleRec(rl.NewRectangle(0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())), terminalBG)
				rl.DrawTextRec(font, "CMD Terminal", rl.NewRectangle(10, 10, float32(rl.GetScreenWidth() - 20), float32(rl.GetScreenHeight() - 40)), 20, spacing, true, session.Color)
				em, command = rl.GuiTextBox(rl.NewRectangle(0, float32(rl.GetScreenHeight() - 30), float32(rl.GetScreenWidth()), 30), command, 256, true)
				
				if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
					if command == "exit" {
						toggleTerminal(fontHeader, font)
					}
				}
				
				break
				
			default:
				windowIndex = 0
			}
		}

		rl.EndDrawing()

	}
	
	if wallet != nil {
		closeWallet()
	}
	
	rl.CloseWindow()
	os.Exit(0)
}

func closeWallet() {
	session.RPCServer = false
	wallet.Stop_RPC_Server()
	wallet.SetOfflineMode()
	session.Syncing = false
	wallet.Close_Encrypted_Wallet()
	session.Path = ""
	wallet = nil
	statusColor = rl.White
	statusText = "Ready"
	windowIndex = 2.0
	progressNow = 0
	start = 0
}

func syncStatus() {
	if wallet != nil {
		wHeight = uint64(wallet.Get_TopoHeight())
		dHeight = uint64(wallet.Daemon_TopoHeight)
		
		if dHeight == 0 {
			statusColor = rl.Red
			statusText = "Status:  [ " + strconv.FormatUint(wHeight, 10) + " / " + strconv.FormatUint(dHeight, 10) + " ]      Attempting connection to " + session.Mode + " node:  " + session.Daemon
		} else if (dHeight - wHeight <= 5) {
			statusColor = session.Color
			statusText = "Status:  [ " + strconv.FormatUint(wHeight, 10) + " / " + strconv.FormatUint(dHeight, 10) + " ]      Connected to " + session.Mode + " node:  " + wallet.Daemon_Endpoint
			progressNow = (float32(wHeight)/float32(dHeight)) * float32(rl.GetScreenWidth())
		} else {
			statusColor = rl.White
			statusText = "Status:  [ " + strconv.FormatUint(wHeight, 10) + " / " + strconv.FormatUint(dHeight, 10) + " ]      Connected to " + session.Mode + " node:  " + wallet.Daemon_Endpoint
			progressNow = (float32(wHeight)/float32(dHeight)) * float32(rl.GetScreenWidth())		
		}
		
		if session.Mode == "offline" {
			statusColor = rl.White
			statusText = "Status:  [ " + strconv.FormatUint(wHeight, 10) + " / 0 ]      OFFLINE MODE"
			progressNow = 0
		}
	} else {
		progressNow = 0
		statusColor = rl.White
		statusText = "Ready"
	}
}

func toggleSidebar() {
	rl.DrawRectangle(rl.GetScreenWidth() - 400, 0, 400, rl.GetScreenHeight() - 30, rl.Black)
}

func toggleMode() {
	statusText = "Switching network mode..."
	
	if session.Mode == "local" {
		session.Mode = "remote"
		
		if session.Network == "Mainnet" {
			session.Daemon = config.RemoteDaemon
			wallet.SetDaemonAddress(config.RemoteDaemon)
		} else {
			session.Daemon = config.RemoteDaemonTestnet
			wallet.SetDaemonAddress(config.RemoteDaemonTestnet)
		}
	} else {
		session.Mode = "local"
		
		if session.Network == "Mainnet" {
			session.Daemon = config.LocalDaemon
			wallet.SetDaemonAddress(config.LocalDaemon)
		} else {
			session.Daemon = config.LocalDaemonTestnet
			wallet.SetDaemonAddress(config.LocalDaemonTestnet)
		}
	}
	
	session.RPCServer = false
	wallet.Stop_RPC_Server()
	wallet.SetOfflineMode()
	session.Syncing = false
	wallet.Close_Encrypted_Wallet()
	wallet = nil
	windowIndex = 2.1
}

// Rescan wallet
func rescan_bc(wallet *walletapi.Wallet, height uint64) {
	wallet.Clean()
	wallet.Rescan_From_Height(0)
}

// Provide the mask for a password
func passwordMask(password string) (bool, string) {
	runes := []rune(password)
	
	if len(runes) == 0 {
		return false, ""
	}
	
	mask := ""
	for i := 0; i < len(runes); i++ {
		mask += "*"
	}
	
	return true, mask
}

// Mask text (UI fix)
func textMask(text string) (bool, string) {
	runes := []rune(text)
	
	if len(runes) == 0 {
		return false, ""
	}
	
	mask := ""
	for i := 0; i < len(runes); i++ {
		mask += string(runes[i])
	}
	
	return true, mask
}

// Convert a string to uint64
func stoUint64(temp string) uint64 {
	conversion, _ := strconv.ParseUint(string(temp), 10, 64)
	
	return conversion
}

// Display seed to the user in his/her preferred language
func display_seed(wallet *walletapi.Wallet) string {
	seed := wallet.GetSeed()
	
	return seed
}

// Display spend key
func display_spend_key(wallet *walletapi.Wallet) (secret crypto.Key, public crypto.Key) {

	keys := wallet.Get_Keys()
	if !account.ViewOnly {
		secret = keys.Spendkey_Secret
	}
	public = keys.Spendkey_Public
	
	return secret, public
}

// Display view key
func display_view_key(wallet *walletapi.Wallet) (secret crypto.Key, public crypto.Key) {

	keys := wallet.Get_Keys()
	secret = keys.Viewkey_Secret
	public = keys.Viewkey_Public
	
	return secret, public
}

// Display wallet view only Keys to create a watchable view only mode
func display_viewwallet_key(wallet *walletapi.Wallet) (view string) {

	view = wallet.GetViewWalletKey()
	
	return view
}

// Trim wallet address
func trim_address(address string) string {
	length := len(address)
	prefix := address[0:4]
	last := address[(length - 16):length]
	trim := prefix + "..." + last
	return trim
}

func toggleTerminal(large rl.Font, small rl.Font) {
	if windowIndex == 5.0 {
		rl.GuiSetFont(large)
		command = ""
		windowIndex = 0
	} else {
		rl.GuiSetFont(small)
		command = ""
		windowIndex = 5.0	
	}
}

func openURL(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		log.Warnf("Error opening URL: %s", err)
	}

}

// handles the output after building tx, takes feedback, confirms or relays tx
func build_relay_transaction(tx *transaction.Transaction, inputs []uint64, input_sum uint64, change uint64, err error, offline_tx bool, amount_list []uint64) bool {

	if err != nil {
		log.Warnf("Error building transaction: %s", err)
		return false
	}
	
	transfer.Inputs = append(transfer.Inputs, uint64(len(inputs)))
	transfer.InputSum = input_sum
	transfer.Change = change
	transfer.Size = float32(len(tx.Serialize()))/1024.0
	transfer.Fees = tx.RctSignature.Get_TX_Fee()
	transfer.TX = tx
	
	amount := uint64(0)
	
	for i := range amount_list {
		amount += amount_list[i]
	}

	if input_sum != (amount + change + tx.RctSignature.Get_TX_Fee()) {
		return false
	}
	
	return true
}

func resetTransfer() {
	transfer.rAddress = nil
	transfer.PaymentID = nil
	transfer.Amount = 0
	transfer.Fees = 0
	transfer.TX = nil
	//transfer.TXID = transfer.TX.GetHash()
	transfer.Size = 0
	transfer.Status = ""
	transfer.Inputs = nil
	transfer.InputSum = 0
	transfer.Change = 0
	transfer.Relay = false
	transfer.OfflineTX = false
	transfer.Filename = ""
	loginPassword = ""
	amountString = ""
	pidString = ""
	receiverString = ""
	checked = false
}

func clearVars() {
	createAccountFilename = ""
	restoreFilename = ""
	createAccountPassword = ""
	restorePassword = ""
	restoreHex = ""
	tempPassword = ""
	seed = ""
	address_s = ""
	passwordError = false
	fileError = false
	createAccountCompleted = false
}

func reloadConfig() {
	session.Network = config.DefaultNetwork
	session.RPCAddress = config.RPCAddress
	session.RPCAuth = config.RPCAuth
	
	if config.DefaultNetwork == "Mainnet" {
		if session.Mode == "remote" {
			session.Daemon = config.RemoteDaemon
		} else {
			session.Daemon = config.LocalDaemon
		}
		
		session.Color = cmdGreen
		globals.Arguments["--testnet"] = false
	} else {
		if session.Mode == "remote" {
			session.Daemon = config.RemoteDaemonTestnet
		} else {
			session.Daemon = config.LocalDaemonTestnet
		}
		
		session.Color = cmdBlue
		globals.Arguments["--testnet"] = true
	}
	
	globals.Initialize()
}

/*
func choose_seed_language(choice int) string {
	languages := mnemonics.Language_List()

	for i := range languages {
		fmt.Fprintf(l.Stderr(), "\033[1m%2d:\033[0m %s\n", i, languages[i])
	}

	if s, err := strconv.Atoi(choice); err == nil {
		choice = s
	}

	for i := range languages { // if user gave any wrong or ot of range choice, choose english
		if choice == i {
			return languages[choice]
		}
	}
	// if no match , return English
	return "English"
}
*/

/*
// Word-wrapping for raylib textboxes based on line length
func wordWrap(text string, length int) ([]string, int) {
	tmpText := text
					
	var segments []string
	runes := []rune(tmpText)
						
	if len(runes) == 0 {
		return nil, 0
	}
						
	for i := 0; i < len(runes); i += length {
		n := i + length
		if n > len(runes) {
			n = len(runes)
		}
		segments = append(segments, string(runes[i:n]))
	}

	return segments, len(segments)
}
*/