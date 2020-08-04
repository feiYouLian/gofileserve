package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//Result Result
type Result struct {
	State int         `json:"state"`
	URL   interface{} `json:"data"`
	Msg   string      `json:"msg"`
}

const (
	assetsDir       = "assets/"
	address         = ":8080"
	staticURLPrefix = "/static/"
)

func main() {

	fs := http.FileServer(http.Dir(assetsDir))
	http.Handle(staticURLPrefix, http.StripPrefix(staticURLPrefix, fs))
	debugPrint("%-6s %-25s --> %s \n", "GET", staticURLPrefix, "获取文件资源")

	http.HandleFunc("/upload", uploadHandle)
	debugPrint("%-6s %-25s --> %s \n", "POST", "/upload", "上传文件（file）")

	debugPrint("Listening and serving HTTP on %s\n", address)
	http.ListenAndServe(address, nil)
}

func uploadHandle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		log.Printf("request method not suppot: %s\n", r.Method)
		return
	}
	// 根据字段名获取表单文件
	formFile, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Get form file failed: %s\n", err)
		return
	}
	defer formFile.Close()
	//创建上传目录
	dateDir := checkAndMakeDateDir()
	// 创建保存文件
	destPath := filepath.Join(assetsDir, dateDir, header.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("Create failed: %s\n", err)
		return
	}
	defer destFile.Close()
	// 读取表单文件，写入保存文件
	_, err = io.Copy(destFile, formFile)
	if err != nil {
		log.Printf("Write file failed: %s\n", err)
		return
	}
	json.NewEncoder(w).Encode(Result{State: 200, URL: staticURLPrefix + dateDir + "/" + header.Filename, Msg: "SUCCESS"})
}

// func uploadMore(w http.ResponseWriter, r *http.Request) {
// 	//设置内存大小
// 	r.ParseMultipartForm(32 << 20)
// 	//获取上传的文件组
// 	files := r.MultipartForm.File["file"]
// 	len := len(files)
// 	for i := 0; i < len; i++ {
// 		//打开上传文件
// 		file, err := files[i].Open()
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		defer file.Close()
// 		//创建上传目录
// 		dateDir := checkAndMakeDateDir()
// 		//创建上传文件
// 		cur, err := os.Create(filepath.Join(assetsDir, dateDir, files[i].Filename))
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		defer cur.Close()
// 		io.Copy(cur, file)
// 	}
// }

func checkAndMakeDateDir() string {
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(assetsDir, dateDir)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	}
	return dateDir
}

func debugPrint(format string, values ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format, values...)
}
