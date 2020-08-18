package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

//Result Result
type Result struct {
	State int         `json:"code"`
	Data  interface{} `json:"data,omitempty"`
	Msg   string      `json:"msg"`
}

const (
	address           = ":8080"
	assetsDir         = "assets/"
	staticURLPrefix   = "/static/"
	formFileName      = "file"
	headerSubdirName  = "subdir"
	username          = "admin"
	password          = "a123456"
	tokenExpireLength = 7 * 24 * time.Hour
	secretKey         = "secretXXXX"
)

func main() {

	fs := http.FileServer(http.Dir(assetsDir))
	http.Handle(staticURLPrefix, http.StripPrefix(staticURLPrefix, fs))
	debugPrint("%-6s %-25s --> %s \n", "GET", staticURLPrefix, "获取文件资源")

	http.HandleFunc("/login", loginHandle)
	debugPrint("%-6s %-25s --> %s  %s: %s\n", "POST", "/login", "用户登录", username, password)

	http.Handle("/upload", oauthValidateMiddleware(http.HandlerFunc(uploadHandle)))
	debugPrint("%-6s %-25s --> %s  %s \n", "POST", "/upload", "上传文件", formFileName)

	debugPrint("Listening and serving HTTP on %s\n", address)
	http.ListenAndServe(address, nil)
}

func loginHandle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		log.Printf("request method not suppot: %s\n", r.Method)
		doRespJSON(w, &Result{State: 400, Msg: "request method not suppot"})
		return
	}

	r.ParseForm()
	username2 := r.PostFormValue("username")
	password2 := r.PostFormValue("password")

	if username2 != username || password2 != password {
		doRespJSON(w, &Result{State: 401, Msg: "username or password error"})
		return
	}

	expire := time.Now().Add(tokenExpireLength).Unix() * 1000
	token, err := generateToken(username, tokenExpireLength)
	if err != nil {
		doRespJSON(w, &Result{State: 500, Msg: "generateToken error"})
		return
	}

	doRespJSON(w, &Result{State: 200, Data: map[string]interface{}{"token": token, "expire": expire}, Msg: "SUCCESS"})
}

func generateToken(username string, expireDuration time.Duration) (string, error) {
	expire := time.Now().Add(expireDuration)
	// 将 uid，用户角色， 过期时间作为数据写入 token 中
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Id:        username,
		ExpiresAt: expire.Unix(),
	})
	// SecretKey 用于对用户数据进行签名，不能暴露
	return token.SignedString([]byte(secretKey))
}

func oauthValidateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accesstoken := r.Header.Get("accesstoken")
		token, err := jwt.ParseWithClaims(accesstoken, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			// 表单文件不关闭 不能正常返回
			formFile, _, err := r.FormFile(formFileName)
			if err != nil {
				log.Printf("Get form file failed: %s\n", err)
				doRespJSON(w, &Result{State: 500, Msg: "Get form file failed"})
				return
			}
			defer formFile.Close()

			log.Printf("accesstoken not pass: %s\n", accesstoken)
			doRespJSON(w, &Result{State: 401, Msg: "accesstoken not pass"})
			return
		}
		claims, ok := token.Claims.(*jwt.StandardClaims)
		if !ok || !token.Valid {
			// 表单文件不关闭 不能正常返回
			formFile, _, err := r.FormFile(formFileName)
			if err != nil {
				log.Printf("Get form file failed: %s\n", err)
				doRespJSON(w, &Result{State: 500, Msg: "Get form file failed"})
				return
			}
			defer formFile.Close()

			log.Printf("accesstoken not pass: %s\n", accesstoken)
			doRespJSON(w, &Result{State: 401, Msg: "accesstoken not pass"})
			return
		}
		debugPrint("username: %-3s", claims.Id)
		next.ServeHTTP(w, r)
	})
}

func uploadHandle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		log.Printf("request method not suppot: %s\n", r.Method)
		doRespJSON(w, &Result{State: 400, Msg: "request method not suppot"})
		return
	}
	// 根据字段名获取表单文件
	formFile, header, err := r.FormFile(formFileName)
	if err != nil {
		log.Printf("Get form file failed: %s\n", err)
		doRespJSON(w, &Result{State: 500, Msg: "Get form file failed"})
		return
	}
	defer formFile.Close()

	//子目录
	subdir := r.Header.Get(headerSubdirName)
	//检查（创建）文件目录
	dir, subURLPrefix := checkAndMakeDir(subdir)

	//文件名 中文转码
	filename, _ := url.QueryUnescape(header.Filename)
	// 创建保存文件
	destPath := filepath.Join(dir, filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("Create file failed: %s\n", err)
		doRespJSON(w, &Result{State: 500, Msg: "Create file failed"})
		return
	}
	defer destFile.Close()
	// 读取表单文件，写入保存文件
	_, err = io.Copy(destFile, formFile)
	if err != nil {
		log.Printf("Write file failed: %s\n", err)
		doRespJSON(w, &Result{State: 500, Msg: "Write file failed"})
		return
	}
	doRespJSON(w, &Result{State: 200, Data: staticURLPrefix + subURLPrefix + "/" + header.Filename, Msg: "SUCCESS"})
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

func checkAndMakeDir(subdir string) (string, string) {
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(assetsDir, subdir, dateDir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}
	subURLPrefix := strings.ReplaceAll(filepath.Join(subdir, dateDir), string(filepath.Separator), "/")
	return dir, subURLPrefix
}

func debugPrint(format string, values ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format, values...)
}

func doRespJSON(w http.ResponseWriter, r *Result) {
	w.Header().Set("content-type", "text/json")
	// msg, _ := json.Marshal(*r)
	// w.Write(msg)
	json.NewEncoder(w).Encode(*r)
}
