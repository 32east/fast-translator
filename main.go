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
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const packetNotFound = "exec: \"xclip\": executable file not found in $PATH"

var _hotkey = hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyC)
var selectedLanguage, selectedModel, prompt string
var languageCheckBoxes []*systray.MenuItem
var modelsCheckBoxes []*systray.MenuItem
var lastClipboard []byte

type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var s = display.Russian.Languages()
var defaultLanguage = s.Name(language.MustParse("ru_RU"))

var Languages = []Language{
	{Code: "en_US", Name: s.Name(language.MustParse("en_US"))}, // Английский
	{Code: "es_ES", Name: s.Name(language.MustParse("es_ES"))}, // Испанский
	{Code: "zh_CN", Name: s.Name(language.MustParse("zh_CN"))}, // Китайский (упрощённый)
	{Code: "fr_FR", Name: s.Name(language.MustParse("fr_FR"))}, // Французский
	{Code: "de_DE", Name: s.Name(language.MustParse("de_DE"))}, // Немецкий
	{Code: "ru_RU", Name: s.Name(language.MustParse("ru_RU"))}, // Русский
	{Code: "pt_PT", Name: s.Name(language.MustParse("pt_PT"))}, // Португальский
	{Code: "ar_SA", Name: s.Name(language.MustParse("ar_SA"))}, // Арабский (Саудовская Аравия)
	{Code: "ja_JP", Name: s.Name(language.MustParse("ja_JP"))}, // Японский
	{Code: "hi_IN", Name: s.Name(language.MustParse("hi_IN"))}, // Хинди
	{Code: "ko_KR", Name: s.Name(language.MustParse("ko_KR"))}, // Корейский
	{Code: "it_IT", Name: s.Name(language.MustParse("it_IT"))}, // Итальянский
	{Code: "tr_TR", Name: s.Name(language.MustParse("tr_TR"))}, // Турецкий
	{Code: "nl_NL", Name: s.Name(language.MustParse("nl_NL"))}, // Голландский
	{Code: "pl_PL", Name: s.Name(language.MustParse("pl_PL"))}, // Польский
	{Code: "uk_UA", Name: s.Name(language.MustParse("uk_UA"))}, // Украинский
	{Code: "vi_VN", Name: s.Name(language.MustParse("vi_VN"))}, // Вьетнамский
	{Code: "th_TH", Name: s.Name(language.MustParse("th_TH"))}, // Тайский
	{Code: "he_IL", Name: s.Name(language.MustParse("he_IL"))}, // Иврит
	{Code: "id_ID", Name: s.Name(language.MustParse("id_ID"))}, // Индонезийский
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
		var findTabSpaces = strings.Index(str, "\n\n\n")

		if findTabSpaces == -1 {
			break
		}

		str = strings.Replace(str, "\n\n\n", "\n\n", -1)
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
	if selectedLanguage == "" {
		selectedLanguage = defaultLanguage
	}

	prompt = `
You are not supposed to write anything except for the translation of the given text.
Preserve absolutely all characters, letter writing style, symbols, font, punctuation marks, letters - EVERYTHING must be preserved!
That is, if a person writes only in lowercase letters, then preserve the style.
If there is an unintelligible symbol - skip it.
Translate into:

If the language of the given text is ` + selectedLanguage + ` → translate to English.
Otherwise, translate to ` + selectedLanguage + `.
`
}

type InputCheckboxes struct {
	Array           []string
	CheckboxesArray []*systray.MenuItem
	DefaultVariable string
	Variable        *string
	OnCheck         func(variable string)
}

func createCheckboxes(input *InputCheckboxes) {
	for _, lang := range input.Array {
		var checkBox = systray.AddMenuItemCheckbox(lang, lang, false)
		input.CheckboxesArray = append(input.CheckboxesArray, checkBox)

		go func() {
			for {
				<-checkBox.ClickedCh

				for _, checkbox := range input.CheckboxesArray {
					checkbox.Uncheck()
				}

				checkBox.Check()
				input.OnCheck(lang)
			}
		}()

		if lang == input.DefaultVariable {
			checkBox.Check()
		}
	}
}

func initLanguageSelector() {
	languageCheckBoxes, modelsCheckBoxes = []*systray.MenuItem{}, []*systray.MenuItem{}

	var mdls, err = GetAvailableModels()
	var availableModels []string
	var count = 0

	if err == nil {
		for _, model := range mdls {
			for _, input := range model.OutputModalities {
				if input == "text" {
					availableModels = append(availableModels, model.Name)
					count++
					break
				}
			}

			if count > 5 {
				break
			}
		}
	} else {
		time.Sleep(time.Second * 5)
		initLanguageSelector()
		return
	}

	createCheckboxes(&InputCheckboxes{
		Array:           availableModels,
		CheckboxesArray: modelsCheckBoxes,
		DefaultVariable: defaultModel,
		OnCheck: func(model string) {
			selectedModel = model
		},
		Variable: &selectedModel,
	})

	systray.AddSeparator()

	var arrLanguages []string
	for _, normalName := range Languages {
		arrLanguages = append(arrLanguages, normalName.Name)
	}

	createCheckboxes(&InputCheckboxes{
		Array:           arrLanguages,
		CheckboxesArray: languageCheckBoxes,
		DefaultVariable: defaultLanguage,
		OnCheck: func(language string) {
			for _, value := range Languages {
				if value.Name == language {
					selectedModel = value.Code
					break
				}
			}

			initLangs()
		},
	})
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
		var ok bool
		lastClipboard, ok = <-clipboardChannel
		if !ok {
			break
		}
	}
	startClipboardWatcher()
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
