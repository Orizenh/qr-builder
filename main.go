package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"image/color"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/skip2/go-qrcode"
)

const DIRECTORY string = "public"
const DEFAULT_BASE64 string = "data:image/svg;base64,iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN+AAABW0lEQVR42uyYMc7sIAyEHVG45AgchavlaByFI1C6QMyTgexLVtr0P2aKCEVfZdlmGNra2tp6E7qqEyIuRD6H8edcC2j6cdVB5o+Q9RvNAYeWqfazR1EgAMkq4IQhvpB1YBYKyIsCYy6IiIV8+Tk4qwOfPcnCxZeXRbo2MDUBIP+4HVcH2oHmgMrCGIXKFFPEuRYAoI0TS18RSlBMZAwgOppDdWMais9zQVgD2nXQjgFKQKaIZA44tGFcJYZ0m6RrMsXbnlwDuEwxhGXYA2gVojXggBolVBrXok5FSPeGMAI03Q/VVRbq9kArlZ4+ygSglaLeL/o68Li0GjBHQ5EeDpD2Q4rWgMsV479NCo89aQS4QpLuitUlaaEo2gPmq5m7S6IxOelmF5cCeh0UGOEAcBoFdC50MALy17VoBPiEJDMi7cC3bV4feGTmIxx8CdX/LLC1tWVR/wIAAP//NsU3kT5oyjoAAAAASUVORK5CYII="
const DEFAULT_URL string = "/api/generate?data=https://www.google.com&size=256&color=000000&bg=ffffff&return=base64"
const DEFAULT_VALUE string = "https://www.google.com"

var templates *template.Template

func init() {
	godotenv.Load()
	templates = template.Must(template.New("index").ParseFiles(
		"templates/index.html",
		"templates/partials/copyrights.html",
		"templates/partials/form.html",
		"templates/partials/header.html",
		"templates/partials/wave.html",
		"templates/partials/qrcode.html",
	))
	for _, templ := range templates.Templates() {
		fmt.Println("Loaded template", templ.Name())
	}
}

func main() {
	port := "3000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	fmt.Println("Service is running on port:", port)
	http.Handle("/"+DIRECTORY+"/", http.StripPrefix("/"+DIRECTORY+"/", http.FileServer(http.Dir(DIRECTORY))))
	http.HandleFunc("/", Index)
	http.HandleFunc("/api/generate", GenerateQR)
	http.ListenAndServe(":"+port, nil)
}

func hostIsAuth(r *http.Request) bool {
	origin := r.Header.Get("origin")
	if origin != "" && strings.Contains(origin, os.Getenv("URL")) {
		return true
	}
	referer := r.Header.Get("referer")
	if referer != "" && strings.Contains(referer, os.Getenv("URL")) {
		return true
	}
	return false
}

func RenderPartial(name string, data any) (string, error) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func SetHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "max-age=31536000")
	w.Header().Set("Accept-Encoding", "gzip, compress, br")
}

func Index(w http.ResponseWriter, r *http.Request) {
	SetHeaders(w)
	err := templates.ExecuteTemplate(w, "index", map[string]any{
		"Date":         time.Now().Format("2006"),
		"Name":         "ORIZENH",
		"Website":      "https://www.orizenh.com",
		"DefaultValue": DEFAULT_VALUE,
		"Base64Image":  template.URL(DEFAULT_BASE64),
		"Path":         template.URL(os.Getenv("URL") + DEFAULT_URL),
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
}

func HexToRgba(hex string) (color.RGBA, error) {
	val, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.RGBA{}, err
	}
	red := uint8(val >> 16 & 0xFF)
	green := uint8(val >> 8 & 0xFF)
	blue := uint8(val & 0xFF)
	return color.RGBA{R: red, G: green, B: blue, A: 255}, nil
}

func GenerateQR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not authorized"}`, http.StatusMethodNotAllowed)
		return
	}

	if !hostIsAuth(r) {
		http.Error(w, `{error: Unauthorized}`, http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	data := r.Form.Get("data")
	path := os.Getenv("URL") + "/api/generate?data=" + data
	if data == "" {
		http.Error(w, `{"error":"You have to fill the field"}`, http.StatusBadRequest)
		return
	}
	size := 256
	if r.Form.Get("size") != "" {
		size, _ = strconv.Atoi(r.Form.Get("size"))
	}
	if size > 512 {
		http.Error(w, `{"error":"Don't be that guy..."}`, http.StatusBadRequest)
		return
	}
	path += "&size=" + strconv.Itoa(size)
	imageName := uuid.New().String() + ".png"
	colorMain := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	colorSecondary := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	var err error
	colorRAW := "000000"
	if r.Form.Get("color") != "" {
		colorRAW = strings.Replace(r.Form.Get("color"), "#", "", -1)
		colorMain, err = HexToRgba(colorRAW)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	}
	path += "&color=" + colorRAW
	bgRAW := "ffffff"
	if r.Form.Get("bg") != "" {
		bgRAW = strings.Replace(r.Form.Get("bg"), "#", "", -1)
		colorSecondary, err = HexToRgba(bgRAW)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}
	path += "&bg=" + bgRAW
	err = qrcode.WriteColorFile(data, qrcode.Medium, size, colorSecondary, colorMain, imageName)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	fileData, _ := os.ReadFile(imageName)
	base64Image := "data:image/svg;base64," + base64.StdEncoding.EncodeToString([]byte(fileData))
	os.Remove(imageName)
	if r.Form.Get("return") == "base64" {
		w.Write([]byte(base64Image))
		return
	}
	if r.Form.Get("return") == "image" {
		w.Header().Set("Content-type", "image/png")
		w.Write([]byte(fileData))
		return
	}
	content, err := RenderPartial("qrcode", map[string]template.URL{
		"Path":        template.URL(path),
		"Base64Image": template.URL(base64Image),
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	result, _ := json.Marshal(map[string]string{
		"content": content,
	})
	w.Write(result)
}
