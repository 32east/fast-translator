package main

import (
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"golang.design/x/clipboard"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	"log"
	"os/exec"
)

const packetNotFound = "exec: \"xclip\": executable file not found in $PATH"

// https://text.pollinations.ai/models
const model = "llama"

const apiURL = "https://text.pollinations.ai"
const prompt = `You are not supposed to write anything except for the translation of the given text.
If an error occurs, then write: "An unforeseen error occurred, or the buffer is empty".
Preserve absolutely all characters, letter writing style, symbols, font, punctuation marks, letters - EVERYTHING must be preserved!
That is, if a person writes only in lowercase letters, then preserve the style.
If there is an unintelligible symbol - skip it.
Translate into English.
`

var _hotkey = hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyC)

func handler(data string) {
	log.Println("Пробуем перевести...")
	log.Println(data)
	var translatedText, err = Generate(data)
	if err != nil {
		beeep.Notify(
			"Переводчик",
			"Текст не был переведён, попробуйте ещё раз.\n",
			"assets/success.png",
		)

		return
	}

	clipboard.Write(clipboard.FmtText, []byte(translatedText))

	beeep.Notify(
		"Переводчик",
		"Переведённый текст скопирован в буфер обмена:\n"+translatedText+"\n",
		"assets/success.png",
	)
}

func checkForXClip() {
	var output = exec.Command("xclip").Run()
	if output != nil && output.Error() == packetNotFound {
		log.Println("Пакет XClip не найден, пытаемся установить:")

		output = exec.Command("sudo", "apt", "install", "xclip").Run()
		if output != nil {
			log.Fatalf("Ошибка установки XClip: %s", output.Error())
		}
	}
}

func startKeyboard() {
	var err = _hotkey.Register()
	if err != nil {
		log.Fatalf("Не удалось зарегистрировать ХотКей: %v", err)
		return
	}

	for {
		<-_hotkey.Keydown()
		var o, oErr = exec.Command("xclip", "-o", "-selection", "primary").Output()
		if oErr != nil {
			log.Printf("Произошла ошибка при просмотре выделенного содержимого: %s", oErr.Error())
		}

		handler(string(o))
	}
}

func startClipboard() {
	if err := clipboard.Init(); clipboard.Init() != nil {
		log.Fatalln(err)
	}
}

func start() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Fast-Translator")
	systray.SetTooltip("Fast-Translator")

	mQuit := systray.AddMenuItem("Выйти", "Закрывает приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	startClipboard()
	checkForXClip()
	go mainthread.Init(startKeyboard)
}

func main() {
	systray.Run(start, nil)
}
