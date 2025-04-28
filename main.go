package main

import (
	"fast-translator/cookie"
	"fmt"
	"github.com/emersion/go-autostart"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
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

type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type InputCheckboxes struct {
	Array           []string
	CheckboxesArray []*systray.MenuItem
	DefaultVariable string
	Variable        *string
	OnCheck         func(variable string)
}

var s = display.Russian.Languages()
var defaultLanguage = s.Name(language.MustParse("ru_RU"))

func n(str string) string { return s.Name(language.MustParse(str)) }

var Languages = []Language{
	{Code: "en_US", Name: n("en_US")}, // Английский
	{Code: "es_ES", Name: n("es_ES")}, // Испанский
	{Code: "zh_CN", Name: n("zh_CN")}, // Китайский (упрощённый)
	{Code: "fr_FR", Name: n("fr_FR")}, // Французский
	{Code: "de_DE", Name: n("de_DE")}, // Немецкий
	{Code: "ru_RU", Name: n("ru_RU")}, // Русский
	{Code: "pt_PT", Name: n("pt_PT")}, // Португальский
	{Code: "ar_SA", Name: n("ar_SA")}, // Арабский (Саудовская Аравия)
	{Code: "ja_JP", Name: n("ja_JP")}, // Японский
	{Code: "hi_IN", Name: n("hi_IN")}, // Хинди
	{Code: "ko_KR", Name: n("ko_KR")}, // Корейский
	{Code: "it_IT", Name: n("it_IT")}, // Итальянский
	{Code: "tr_TR", Name: n("tr_TR")}, // Турецкий
	{Code: "nl_NL", Name: n("nl_NL")}, // Голландский
	{Code: "pl_PL", Name: n("pl_PL")}, // Польский
	{Code: "uk_UA", Name: n("uk_UA")}, // Украинский
	{Code: "vi_VN", Name: n("vi_VN")}, // Вьетнамский
	{Code: "th_TH", Name: n("th_TH")}, // Тайский
	{Code: "he_IL", Name: n("he_IL")}, // Иврит
	{Code: "id_ID", Name: n("id_ID")}, // Индонезийский
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
		fmt.Sprintf("Переводчик (%s)", selectedModel),
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

	if err := pasteToClipboard(translatedText); err != nil {
		notify(&Notify{
			Message: fmt.Sprintf("Произошла ошибка вставки в буфер обмена: %s", err.Error()),
			Icon:    "failed",
		})

		return
	}

	notify(&Notify{
		Message: "Переведённый текст скопирован в буфер обмена:\n" + translatedText,
		Icon:    "success",
	})
}

func pasteToClipboard(text string) error {
	var cmd = exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func checkClipboard(selection string) (string, error) {
	var o, oErr = exec.Command("xclip", "-o", "-selection", selection).Output()
	if oErr != nil {
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
	if err := _hotkey.Register(); err != nil {
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
			notify(&Notify{
				Message: fmt.Sprintf("Произошла ошибка при просмотре выделенного содержимого: %s", strErr.Error()),
				Icon:    "failed",
			})

			continue
		}

		handler(formatString(str))
	}
}

func initLangs() {
	if selectedLanguage == "" {
		if cookieLang := cookie.Get("selected_language"); cookieLang != nil {
			selectedLanguage = cookieLang.(string)
		} else {
			selectedLanguage = defaultLanguage
		}
	}

	if cookieModel := cookie.Get("selected_model"); cookieModel != nil {
		selectedModel = cookieModel.(string)
	} else {
		selectedModel = defaultModel
	}

	prompt = fmt.Sprintf(`
You are a translation assistant. Your task is to translate the input text in either direction between %s and English:

1. Detect the language of the input text.
2. If it is in %s, translate it into English.
3. Otherwise, translate it into %s.

Output only the translated text—do NOT add any explanations, comments or markup.
Preserve absolutely all characters, casing, punctuation, spaces, line breaks and symbols exactly as in the original.
If you encounter any unintelligible or garbled symbol, include it unchanged in the output.
`, selectedLanguage, selectedLanguage, selectedLanguage)
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

	if err != nil {
		time.Sleep(time.Second * 5)
		initLanguageSelector()
		return
	} else {
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
	}

	var cookieModel = cookie.Get("selected_model")
	if cookieModel == nil {
		cookieModel = defaultModel
	}

	createCheckboxes(&InputCheckboxes{
		Array:           availableModels,
		CheckboxesArray: modelsCheckBoxes,
		DefaultVariable: cookieModel.(string),
		OnCheck: func(model string) {
			selectedModel = model
			cookie.Set("selected_model", model)
		},
		Variable: &selectedModel,
	})

	systray.AddSeparator()

	var arrLanguages []string
	for _, normalName := range Languages {
		arrLanguages = append(arrLanguages, normalName.Name)
	}

	var cookieLanguage = cookie.Get("selected_language")
	if cookieLanguage == nil {
		cookieLanguage = defaultLanguage
	}

	createCheckboxes(&InputCheckboxes{
		Array:           arrLanguages,
		CheckboxesArray: languageCheckBoxes,
		DefaultVariable: cookieLanguage.(string),
		OnCheck: func(language string) {
			for _, value := range Languages {
				if value.Name == language {
					selectedLanguage = value.Code
					cookie.Set("selected_language", value.Name)
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
	cookie.Initialize()
	initLangs()

	systray.SetIcon(icon.Data)
	systray.SetTitle("Fast-Translator")
	systray.SetTooltip("Fast-Translator")

	initAutostart()
	initLanguageSelector()
	systray.AddSeparator()
	initExitButton()

	for i := 1; i <= 5; i++ {
		if _, xclipErr := exec.Command("xclip").Output(); xclipErr != nil {
			if i >= 5 {
				notify(&Notify{
					Message: fmt.Sprintf("Похоже, что XClip не установлен: %s", xclipErr.Error()),
					Icon:    "failed",
				})

				os.Exit(0)
			}

			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	go mainthread.Init(startKeyboard)
}

func init() {
	runtime.GOMAXPROCS(1)
}

func main() {
	systray.Run(start, nil)
}
