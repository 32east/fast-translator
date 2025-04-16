package main

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"golang.design/x/clipboard"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	"log"
	"os/exec"
	"strings"
)

// https://text.pollinations.ai/models
const model = "llama"
const nativeLanguage = "Русский"
const packetNotFound = "exec: \"xclip\": executable file not found in $PATH"
const apiURL = "https://text.pollinations.ai"

var _hotkey = hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyC)
var defaultNativeLanguage, prompt string
var checkboxes []*systray.MenuItem

var Languages = []string{
	"Английский",
	"Испанский",
	"Китайский",
	"Французский",
	"Немецкий",
	"Русский",
	"Португальский",
	"Арабский",
	"Японский",
	"Хинди",
	"Корейский",
	"Итальянский",
	"Турецкий",
	"Голландский",
	"Польский",
	"Украинский",
	"Вьетнамский",
	"Тайский",
	"Иврит",
	"Индонезийский",
}

type Notify struct {
	Message string `json:"message"`
	Icon    string `json:"icon"`
}

func notify(Notify *Notify) {
	var iconPath string
	if Notify.Icon != "" {
		iconPath = fmt.Sprintf("assets/%s.png", Notify.Icon)
	}

	beeep.Notify(
		"Переводчик",
		Notify.Message,
		iconPath,
	)
}

func handler(data string) {
	log.Printf("Пробуем перевести: %s", data)

	var translatedText, err = Generate(data)
	if err != nil {
		notify(&Notify{
			Message: "Текст не был переведён, попробуйте ещё раз.",
			Icon:    "failed",
		})

		return
	}

	clipboard.Write(clipboard.FmtText, []byte(translatedText))
	notify(&Notify{
		Message: "Переведённый текст скопирован в буфер обмена:\n" + translatedText,
		Icon:    "success",
	})
}

func checkForXClip() {
	var output = exec.Command("xclip").Run()
	if output != nil && output.Error() == packetNotFound {
		log.Println("Пакет XClip не найден, пытаемся установить...")

		output = exec.Command("apt", "install", "xclip").Run()
		if output != nil {
			log.Fatalf("Ошибка установки XClip: %s", output.Error())
		}
	}
}

func startKeyboard() {
	var err = _hotkey.Register()
	if err != nil {
		notify(&Notify{
			Message: fmt.Sprintf("Не удалось зарегистрировать ХотКей: %v", err),
			Icon:    "failed",
		})
		return
	}

	for {
		<-_hotkey.Keydown()
		var o, oErr = exec.Command("xclip", "-o", "-selection", "primary").Output()
		if oErr != nil {
			notify(&Notify{
				Message: fmt.Sprintf("Произошла ошибка при просмотре выделенного содержимого: %s", oErr.Error()),
				Icon:    "failed",
			})

			continue
		}

		var str = string(o)
		if strings.TrimSpace(str) == "" {
			notify(&Notify{
				Message: "Ничего не выделено для перевода.",
				Icon:    "failed",
			})

			continue
		}

		handler(str)
	}
}

func startClipboard() {
	if err := clipboard.Init(); clipboard.Init() != nil {
		notify(&Notify{
			Message: fmt.Sprintf("Не удалось инициализировать сервис буфера обмена: %s", err),
			Icon:    "failed",
		})
	}
}

func initLangs() {
	if defaultNativeLanguage == "" {
		defaultNativeLanguage = nativeLanguage
	}

	prompt = `
You are not supposed to write anything except for the translation of the given text.
Preserve absolutely all characters, letter writing style, symbols, font, punctuation marks, letters - EVERYTHING must be preserved!
That is, if a person writes only in lowercase letters, then preserve the style.
If there is an unintelligible symbol - skip it.
Translate into:

If the language of the given text is ` + defaultNativeLanguage + ` → translate to English.
Otherwise, translate to ` + defaultNativeLanguage + `.
`
	log.Println(prompt)
}

func initLanguageSelector() {
	checkboxes = []*systray.MenuItem{}

	for _, lang := range Languages {
		var checkBox = systray.AddMenuItemCheckbox(lang, lang, false)
		checkboxes = append(checkboxes, checkBox)

		go func() {
			for {
				<-checkBox.ClickedCh

				for _, checkbox := range checkboxes {
					checkbox.Uncheck()
				}

				checkBox.Check()
				defaultNativeLanguage = lang
				initLangs()
			}
		}()

		if lang == nativeLanguage {
			checkBox.Check()
		}
	}
}

func initExitButton() {
	var mQuit = systray.AddMenuItem("Выйти", "Закрывает приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func start() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Fast-Translator")
	systray.SetTooltip("Fast-Translator")

	initLanguageSelector()
	initExitButton()

	startClipboard()
	checkForXClip()
	go mainthread.Init(startKeyboard)
}

func init() {
	initLangs()
}

func main() {
	systray.Run(start, nil)
}
