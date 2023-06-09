/*
 * @Author: Bin
 * @Date: 2023-04-30
 * @FilePath: /webdav-server-debug/main.go
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/webdav"
)

var (
	flagRootDir   = flag.String("dir", "./data/", "webdav root dir")
	flagHttpAddr  = flag.String("http", ":8061", "http or https address")
	flagHttpsMode = flag.Bool("https-mode", false, "use https mode")
	flagCertFile  = flag.String("https-cert-file", "cert.pem", "https cert file")
	flagKeyFile   = flag.String("https-key-file", "key.pem", "https key file")
	flagUserName  = flag.String("user", "", "user name")
	flagPassword  = flag.String("password", "", "user password")
	flagReadonly  = flag.Bool("read-only", false, "read only")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of WebDAV Server\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	fs := &webdav.Handler{
		FileSystem: webdav.Dir(*flagRootDir),
		LockSystem: webdav.NewMemLS(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if *flagUserName != "" && *flagPassword != "" {
			username, password, ok := req.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if username != *flagUserName || password != *flagPassword {
				http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
				return
			}
		}
		if req.Method == "GET" && handleDirList(fs.FileSystem, w, req) {
			return
		}
		if *flagReadonly {
			switch req.Method {
			case "PUT", "DELETE", "PROPPATCH", "MKCOL", "COPY", "MOVE":
				http.Error(w, "WebDAV: Read Only!!!", http.StatusForbidden)
				return
			}
		}
		fs.ServeHTTP(w, req)
	})

	var domain = "http://" + *flagHttpAddr
	serverUrl, err := url.Parse(domain)
	if err == nil && serverUrl != nil {
		if serverUrl.Hostname() == "" {
			serverUrl.Host = "127.0.0.1:" + serverUrl.Port()
		}
		domain = serverUrl.String()
	}
	fmt.Fprintf(os.Stderr, "Start of WebDAV Server to %s \n", domain)

	if *flagHttpsMode {
		http.ListenAndServeTLS(*flagHttpAddr, *flagCertFile, *flagKeyFile, nil)
	} else {
		http.ListenAndServe(*flagHttpAddr, nil)
	}

}

func handleDirList(fs webdav.FileSystem, w http.ResponseWriter, req *http.Request) bool {
	ctx := context.Background()
	f, err := fs.OpenFile(ctx, req.URL.Path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		return false
	}
	dirs, err := f.Readdir(-1)
	if err != nil {
		log.Print(w, "Error reading directory", http.StatusInternalServerError)
		return false
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<pre>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", name, name)
	}
	fmt.Fprintf(w, "</pre>\n")
	return true
}
