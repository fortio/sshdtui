package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/scli"
	"fortio.org/terminal"
	"fortio.org/terminal/ansipixels"
	bj "fortio.org/terminal/blackjack/cli"
	brick "fortio.org/terminal/brick/cli"
	life "fortio.org/terminal/life/cli"
	"fortio.org/terminal/life/conway"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const KeyFile = "./host_key"

type InputAdapter struct {
	ansipixels.InputReader
}

func (ia InputAdapter) RawMode() error {
	return nil
}

func (ia InputAdapter) NormalMode() error {
	return nil
}

func (ia InputAdapter) StartDirect() {
}

const FPS = 30

func RunBrick(ap *ansipixels.AnsiPixels) {
	bc := brick.BrickConfig{
		FPS:      FPS,
		NumLives: 3,
		Ap:       ap,
	}
	bc.Run()
}

func RunGameOfLife(ap *ansipixels.AnsiPixels) {
	game := conway.Game{
		AP:       ap,
		HasMouse: true,
	}
	life.RunGame(&game, 0.1, false)
}

func RunBlackjackGame(ap *ansipixels.AnsiPixels) {
	game := bj.Game{
		AP:          ap,
		Playing:     true,
		State:       bj.StatePlayerTurn,
		Balance:     100,
		Bet:         10,
		BorderColor: ansipixels.Green,
		BorderBG:    ansipixels.GreenBG,
	}
	game.InitDeck(4)
	game.RunGame(ap)
}

func envMap(s ssh.Session, pty ssh.Pty) map[string]string {
	m := make(map[string]string, len(s.Environ()))
	for _, kv := range s.Environ() {
		if before, after, ok := strings.Cut(kv, "="); ok {
			m[before] = after
		}
	}
	if m["TERM"] == "" {
		m["TERM"] = pty.Term
	}
	return m
}

func ResetAP(ap *ansipixels.AnsiPixels, resizeFunc func() error, msg string, args ...any) {
	ap.OnResize = resizeFunc
	ap.ShowCursor()
	ap.MouseTrackingOff()
	_ = ap.OnResize()
	ap.WriteAt(0, ap.H-2, msg, args...)
	ap.EndSyncMode()
}

func Handler(s ssh.Session) {
	p, c, ok := s.Pty()
	env := envMap(s, p)
	log.Infof("New SSH session from %v user=%s, env=%v", s.RemoteAddr(), s.User(), env)
	log.S(log.Info, "Pty:", log.Any("pty", p), log.Any("ok", ok))
	if !ok {
		log.Warnf("No PTY requested, closing session")
		fmt.Fprintln(s, "Need PTY to demo interactive TUI, sorry!")
		return
	}
	width, height := p.Window.Width, p.Window.Height
	ap := &ansipixels.AnsiPixels{
		Out:       bufio.NewWriter(s),
		FPS:       FPS,
		H:         height,
		W:         width,
		C:         make(chan os.Signal, 1),
		ColorMode: ansipixels.DetectColorModeEnv(func(key string) string { return env[key] }),
	}
	fps := 60
	timeout := time.Duration(1000/fps) * time.Millisecond
	ir := terminal.NewTimeoutReader(s, timeout)
	ia := InputAdapter{ir}
	ap.SharedInput = ia
	ap.GetSize = func() error {
		ap.W, ap.H = width, height
		return nil
	}
	ap.SkipOpen = true
	ap.AutoSync = true
	ap.TrueColor = true
	_ = ap.Open()
	resizeFunc := func() error {
		ap.ClearScreen()
		ap.WriteBoxed(ap.H/2-1,
			"Ansipixels sshdtui v%s!\nTerminal width: %d, height: %d\n"+
				"You can resize me!\nQ to quit\n1 for brick game,  \n2 for game of life,\n3 for BlackJack.   ",
			cli.ShortVersion, width, height)
		ap.EndSyncMode()
		return nil
	}
	ap.OnResize = resizeFunc
	_ = ap.OnResize()
	defer func() {
		ap.MouseTrackingOff()
		ap.ShowCursor()
	}()
	keepGoing := true
	for keepGoing {
		select {
		case w := <-c:
			if w.Width == width && w.Height == height {
				continue
			}
			width, height = w.Width, w.Height
			log.LogVf("Window resized to %dx%d", width, height)
			// Only send if it's not already queued
			select {
			case ap.C <- ansipixels.ResizeSignal:
				// signal sent
			default:
				// channel full; nothing to do (will get processed in next ReadOrResizeOrSignalOnce)
			}
		default:
			n, err := ap.ReadOrResizeOrSignalOnce()
			if err != nil {
				log.Errf("Error reading input or resizing or signaling: %v", err)
				keepGoing = false
			}
			if n == 0 {
				continue
			}
			c := ap.Data[0]
			switch c {
			case 3, 'q': // Ctrl-C or 'q'
				log.Infof("Exit requested, closing session")
				ResetAP(ap, resizeFunc, "Exit requested, closing session.")
				keepGoing = false
			case '1':
				log.Infof("Starting Brick game")
				ap.WriteAt(0, ap.H-2, "Starting Brick game...")
				ap.EndSyncMode()
				RunBrick(ap)
				ResetAP(ap, resizeFunc, "Exited Brick game ")
			case '2':
				log.Infof("Starting Game of Life")
				ap.WriteAt(0, ap.H-2, "Starting Game of Life...")
				ap.EndSyncMode()
				RunGameOfLife(ap)
				ResetAP(ap, resizeFunc, "Exited Game of Life ")
			case '3':
				log.Infof("Starting BlackJack game")
				ap.WriteAt(0, ap.H-2, "Starting BlackJack game...")
				ap.EndSyncMode()
				RunBlackjackGame(ap)
				ResetAP(ap, resizeFunc, "Exited BlackJack game ")
			default:
				// echo back
				ap.WriteAt(0, ap.H-2, "Received %q", ap.Data)
				ap.ClearEndOfLine()
			}
		}
		ap.EndSyncMode()
	}
}

func CheckKeyFile(keyFile string) {
	_, err := os.Stat(keyFile)
	if err == nil {
		log.Infof("Using existing key file %s", keyFile)
		return
	}
	if !os.IsNotExist(err) {
		log.Fatalf("%s: %v", keyFile, err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("%v", err)
	}
	privateKeyBlock, err := gossh.MarshalPrivateKey(key, "")
	if err != nil {
		log.Fatalf("%v", err)
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyBlock)
	err = os.WriteFile(keyFile, privateKeyBytes, 0o600)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Warnf("Generated new host key at %s", keyFile)
}

func main() {
	port := flag.String("port", ":2222", "Port/address to listen on")
	scli.ServerMain()
	CheckKeyFile(KeyFile)
	log.Infof("Starting SSH server on %s", *port)
	log.Fatalf("%v", ssh.ListenAndServe(*port, Handler, ssh.HostKeyFile(KeyFile)))
}
