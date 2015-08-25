package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	//"regexp"
	"strings"
)

type FileInfo struct {
	Name           string
	Size           int64
	LinkToDownload string
}

var templt *template.Template

func main() {
	var err1 error
	file := path.Join("html", "disignFile.html")
	if templt, err1 = template.ParseFiles(file); err1 != nil {
		log.Fatal("problem with parsing file", err1)
	}
	fs := http.FileServer(http.Dir("html"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/cloud/", homePage)

	http.HandleFunc("/cloud/usersStorage/", func(w http.ResponseWriter, r *http.Request) {
		a := r.URL.String()
		name, _, successAuth := r.BasicAuth()
		if successAuth {
			b := strings.Split(a, "/")
			cd := "cloud/usersStorage/" + name + "/" + b[len(b)-1]
			http.ServeFile(w, r, cd)

		} else {
			http.Error(w, "protected page", http.StatusForbidden)
		}
	})

	port := flag.String("port", ":9111", "port in server")
	flag.Parse()
	if err4 := http.ListenAndServeTLS(*port, "cert.pem", "key.pem", nil); err4 != nil {
		log.Fatal("failed to start server", err4)
	}
}

func showEntireFolder(writer http.ResponseWriter, request *http.Request, userPath string, temp *template.Template, userName string) {

	userFolderEntire, err := os.Open(userPath)
	if err != nil {
		log.Println(err)
		http.Error(writer, "problem with userpath", http.StatusBadRequest)
		return
	}
	fi, err := userFolderEntire.Readdir(-1)
	if err != nil {
		log.Println(err)
		http.Error(writer, "problem with userpath", http.StatusBadRequest)
		return
	}
	defer userFolderEntire.Close()
	var SliceFolder []FileInfo
	SliceFolder = make([]FileInfo, 0)
	for _, fi := range fi {
		var obj = FileInfo{
			Name:           fi.Name(),
			Size:           fi.Size(),
			LinkToDownload: "/cloud/usersStorage/" + fi.Name(),
		}
		SliceFolder = append(SliceFolder, obj)
	}
	stri := fmt.Sprint(request.Header.Get("Accept"))
	if controlQuerry := strings.Contains(stri, "application/json"); controlQuerry {
		notesJson, erro := json.Marshal(SliceFolder)
		if erro != nil {
			log.Println("Error Json")
			return
		}
		writer.Header().Set("Content-type", "application/json")
		writer.Write(notesJson)
		return
	}
	stri = fmt.Sprint(request.Header.Get("Action"))
	if controlQuerry := strings.Contains(stri, "delete"); controlQuerry {
		a := strings.Fields(stri)
		if err = deleteFile(a[1], userPath); err != nil {
			http.Error(writer, "problem with deleting", http.StatusBadRequest)
		}
		return
	}

	if controlQuerry := strings.Contains(stri, "upload"); controlQuerry {
		a := strings.Fields(stri)
		dest, err := os.Create(userPath + "/" + a[1])
		if err != nil {
			log.Println(err)
			return
		}
		defer dest.Close()
		body := &bytes.Buffer{}
		body.ReadFrom(request.Body)
		request.Body.Close()
		if _, err := io.Copy(dest, body); err != nil {
			log.Println(err)
			return
		}
		return
	}

	temp.Execute(writer, SliceFolder)
}

func deleteFile(nameFile string, path string) error {
	err := os.Remove(path + "/" + nameFile)
	if err != nil {
		log.Println(err, "problem with deleting file")
		return err
	}
	return nil
}

func homePage(writer http.ResponseWriter, request *http.Request) {
	name, _, successAuth := request.BasicAuth()
	if !successAuth {
		writer.Header().Set("WWW-Authenticate", `Basic realm="protectedpage"`)
		http.Error(writer, "bad auth", http.StatusUnauthorized)
		return
	}
	writer.Header().Set("Content-type", "text/html")
	userPath := "cloud/usersStorage/" + name
	err := os.MkdirAll(userPath, 0777)
	if err != nil {
		log.Println(err, "problem with creating user's directory")
		http.Error(writer, "problen with user path", http.StatusBadRequest)
	}

	if reqSend := request.FormValue("sendButton"); reqSend != "" {
		uploadFile(request, userPath)
		showEntireFolder(writer, request, userPath, templt, name)
		return
	}
	if reqSend := request.FormValue("deleteButton"); reqSend != "" {
		slice, found := request.Form["option"]
		for i, _ := range slice {
			if err = deleteFile(slice[i], userPath); err != nil && found {
				http.Error(writer, "problem with deleting", http.StatusBadRequest)
			}
		}
		showEntireFolder(writer, request, userPath, templt, name)
		return
	}
	showEntireFolder(writer, request, userPath, templt, name)
}

func uploadFile(request *http.Request, userPath string) {
	m := request.MultipartForm
	files := m.File["myfiles"]
	for i, _ := range files {
		saveFile(files[i], userPath)
	}
}

func saveFile(fil *multipart.FileHeader, userPath string) {
	file, err := fil.Open()
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	dest, err := os.Create(userPath + "/" + fil.Filename)
	if err != nil {
		log.Println(err)
		return
	}
	defer dest.Close()
	if _, err := io.Copy(dest, file); err != nil {
		log.Println(err)
		return
	}
}
