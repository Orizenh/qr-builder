package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/skip2/go-qrcode"
)

const DIRECTORY string = "public"
const DEFAULT_BASE64 string = "data:image/svg;base64,iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN+AAABW0lEQVR42uyYMc7sIAyEHVG45AgchavlaByFI1C6QMyTgexLVtr0P2aKCEVfZdlmGNra2tp6E7qqEyIuRD6H8edcC2j6cdVB5o+Q9RvNAYeWqfazR1EgAMkq4IQhvpB1YBYKyIsCYy6IiIV8+Tk4qwOfPcnCxZeXRbo2MDUBIP+4HVcH2oHmgMrCGIXKFFPEuRYAoI0TS18RSlBMZAwgOppDdWMais9zQVgD2nXQjgFKQKaIZA44tGFcJYZ0m6RrMsXbnlwDuEwxhGXYA2gVojXggBolVBrXok5FSPeGMAI03Q/VVRbq9kArlZ4+ygSglaLeL/o68Li0GjBHQ5EeDpD2Q4rWgMsV479NCo89aQS4QpLuitUlaaEo2gPmq5m7S6IxOelmF5cCeh0UGOEAcBoFdC50MALy17VoBPiEJDMi7cC3bV4feGTmIxx8CdX/LLC1tWVR/wIAAP//NsU3kT5oyjoAAAAASUVORK5CYII="
const DEFAULT_URL string = "/api/generate?data=https://www.google.com&size=256&color=000000&bg=ffffff&return=base64"
const DEFAULT_VALUE string = "https://www.google.com"

func init() {
	godotenv.Load()
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
	result, _ := json.Marshal(map[string]string{
		"content": getQrCode(base64Image, path),
	})
	w.Write(result)
}

func getQrCode(base64Image string, path string) string {
	return `<a href="` + path + `&return=image" target="_blank">
		<img src="` + base64Image + `" alt="QR Code" />
	</a>
	<textarea id="copy" readonly>` + base64Image + `</textarea>
	<button class="btn" onclick="copyToClipboard('#copy')">Copy base64</button>`
}

func getForm() string {
	return `
	<div class="flex-wrapper">
	<form id="generateQR" method="POST"
    data-path="/api/generate">
		<div class="input-group">
			<label for="data">URL to append to the QR</label>
			<input id="data" name="data" type="text" 
			value="` + DEFAULT_VALUE + `"
			placeholder="example: https://www.googe.com" />
		</div>
		<div class="input-group">
			<label for="color">Main color</label>
			<input id="color" name="color" type="color"
			placeholder="Main color" value="#000000" />
		</div>
		<div class="input-group">
			<label for="bg">Background</label>
			<input id="bg" name="bg" type="color" 
			placeholder="Background color" value="#ffffff" />
		</div>
        <button class="btn" type="submit">Générer le QRCODE</button>
    </form>
	<div id="content">
		` + getQrCode(DEFAULT_BASE64, os.Getenv("URL")+DEFAULT_URL) + `
	</div>
	</div>
`
}

func getHeader() string {
	return `<head>
		<meta charset="utf-8" />
		<link rel="icon" type="image/png" href="./public/images/favicon-96x96.png" sizes="96x96" />
		<link rel="icon" type="image/svg+xml" href="./public/images/favicon.svg" />
		<link rel="shortcut icon" href="./public/images/favicon.ico" />
		<link rel="apple-touch-icon" sizes="180x180" href="./public/images/apple-touch-icon.png" />
		<meta name="apple-mobile-web-app-title" content="QR Builder" />
		<link rel="manifest" href="/site.webmanifest" />
		<meta name="theme-color" content="#4285f4"> 
		<meta name="viewport" content="width=device-width, initial-scale=1" />	
		<link href="./public/css/style.css" rel="stylesheet">
		<title>QR Builder</title>
		<meta name="description" content="QR Builder is a simple tool to generate QR Code" />
		<meta name="keywords" content="QR Code, QR Builder, QR Code Generator" />
		<link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/toastify-js/src/toastify.min.css">
		<script type="text/javascript" src="https://cdn.jsdelivr.net/npm/toastify-js" defer></script>
		<script src="./public/js/functions.js" defer></script>
		<link rel="preconnect" href="https://fonts.googleapis.com">
		<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
		<link href="https://fonts.googleapis.com/css2?family=Roboto:ital,wght@0,100..900;1,100..900&display=swap" rel="stylesheet">
	</head>`
}

func Index(w http.ResponseWriter, r *http.Request) {
	content := `
<!doctype html>
<html lang="en">
	` + getHeader() + `
	<body>
	<svg 
	id="waves"
	viewBox="0 0 2 1" 
	preserveAspectRatio="none">
	 <defs>
	   <path id="w" 
		 d="
		 m0 1v-.5 
		 q.5.5 1 0
		 t1 0 1 0 1 0
		 v.5z" />
	 </defs>
	 <g>
	  <use href="#w" y=".0" fill="#04a118" />
	  <use href="#w" y=".1" fill="#70b529" />
	  <use href="#w" y=".2" fill="#91cc25" />
	 </g>
	</svg>
		<main class="flex center column">
			<h1>QR Builder</h1>
			<img src="./public/images/logo.svg" alt="QR Code" />` + getForm() + `
		</main>
	</body>
	</html>`
	w.Write([]byte(content))
}
