package main

import (
	"context"
	"fmt"
	"github.com/emersion/go-autostart"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"golang.design/x/clipboard"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	"os"
	"os/exec"
	"runtime"
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
var lastClipboard []byte

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
	fmt.Printf("Пробуем перевести: %s", data)

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
	if runtime.GOOS == "windows" {
		return
	}

	var output = exec.Command("xclip").Run()
	if output != nil && output.Error() == packetNotFound {
		fmt.Println("Пакет XClip не найден, пытаемся установить...")

		output = exec.Command("apt", "install", "xclip").Run()
		if output != nil {
			panic(output)
		}
	}
}

func checkClipboard(selection string) (string, error) {
	if runtime.GOOS == "windows" {
		return "", nil
	}

	var o, oErr = exec.Command("xclip", "-o", "-selection", selection).Output()
	if oErr != nil {
		notify(&Notify{
			Message: fmt.Sprintf("Произошла ошибка при просмотре выделенного содержимого: %s", oErr.Error()),
			Icon:    "failed",
		})

		return "", oErr
	}

	return string(o), nil
}

func formatString(str string) string {
	str = strings.Trim(str, "\n")
	str = strings.Trim(str, " ")
	str = strings.Replace(str, "\r", "", -1)

	for {
		var findTabSpaces = strings.Index(str, "\n\n")

		if findTabSpaces == -1 {
			break
		}

		str = strings.Replace(str, "\n\n", "\n", -1)
	}

	return str
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
		var str, strErr = checkClipboard("primary")
		if strErr != nil {
			return
		}

		if strings.TrimSpace(str) == "" {
			str = string(lastClipboard)

			if strings.TrimSpace(str) == "" {
				notify(&Notify{
					Message: "В буфере обмена - пусто.",
					Icon:    "failed",
				})
			}

			continue
		}

		str = formatString(str)
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

func startClipboardWatcher() {
	var clipboardChannel = clipboard.Watch(context.Background(), clipboard.FmtText)
	for {
		lastClipboard = <-clipboardChannel
	}
}

func initAutostart() {
	var executable, err = os.Executable()
	if err != nil {
		notify(&Notify{
			Message: fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %s", err),
			Icon:    "failed",
		})

		return
	}

	var app = &autostart.App{
		Name:        "Fast-Translator",
		DisplayName: "Fast-Translator",
		Exec:        []string{"sh", "-c", executable},
	}

	app.Enable()
}

func start() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Fast-Translator")
	systray.SetTooltip("Fast-Translator")

	initAutostart()
	initLanguageSelector()
	systray.AddSeparator()
	initExitButton()

	startClipboard()

	if runtime.GOOS != "windows" {
		checkForXClip()
	}

	go startClipboardWatcher()
	go mainthread.Init(startKeyboard)
}

func init() {
	runtime.GOMAXPROCS(1)
	initLangs()
}

func main() {
	systray.Run(start, nil)
}
