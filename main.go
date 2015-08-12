package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
)

type FileInfo struct {
	Name           string
	Size           int64
	LinkToDownload string
}

type InfoFile struct {
	Info []FileInfo
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
	http.HandleFunc("/usersStorage/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
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
		log.Fatal(err)
	}

	defer userFolderEntire.Close()
	fi, err := userFolderEntire.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}
	var sliceFolder InfoFile
	sliceFolder.Info = make([]FileInfo, 0)
	for _, fi := range fi {
		var obj = FileInfo{
			Name:           fi.Name(),
			Size:           fi.Size(),
			LinkToDownload: "/" + userPath + "/" + fi.Name(),
		}
		sliceFolder.Info = append(sliceFolder.Info, obj)
	}
	temp.Execute(writer, sliceFolder)
}

func deleteFile(nameFile string, path string) {
	os.Remove(path + "/" + nameFile)
}

func homePage(writer http.ResponseWriter, request *http.Request) {
	name, _, successAuth := request.BasicAuth()
	if !successAuth {
		writer.Header().Set("WWW-Authenticate", `Basic realm="protectedpage"`)
		writer.WriteHeader(401)
		return
	}
	writer.Header().Set("Content-type", "text/html")
	userPath := "usersStorage/" + name
	os.MkdirAll(userPath, 0777)
	request.ParseMultipartForm(0)
	if reqSend := request.FormValue("sendButton"); reqSend != "" {
		uploadFile(request, userPath)
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
	defer file.Close()
	if err != nil {
		log.Println(err)
		return
	}
	dest, err := os.Create(userPath + "/" + fil.Filename)
	defer dest.Close()
	if err != nil {
		log.Println(err)
		return
	}
	if _, err := io.Copy(dest, file); err != nil {
		log.Println(err)
		return
	}
}
