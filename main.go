package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var httpFlag = flag.String("http", ":8080", "Serve HTTP at given address")
var httpsFlag = flag.String("https", "", "Serve HTTPS at given address")
var certFlag = flag.String("cert", "", "Use the provided TLS certificate")
var keyFlag = flag.String("key", "", "Use the provided TLS key")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	if len(flag.Args()) > 0 {
		return fmt.Errorf("too many arguments: %s", flag.Args()[0])
	}

	http.HandleFunc("/", handler)

	if *httpFlag == "" && *httpsFlag == "" {
		return fmt.Errorf("must provide -http and/or -https")
	}
	if (*httpsFlag != "" || *certFlag != "" || *keyFlag != "") && (*httpsFlag == "" || *certFlag == "" || *keyFlag == "") {
		return fmt.Errorf("-https -cert and -key must be used together")
	}

	ch := make(chan error, 2)

	if *httpFlag != "" {
		go func() {
			ch <- http.ListenAndServe(*httpFlag, nil)
		}()
	}
	if *httpsFlag != "" {
		go func() {
			ch <- http.ListenAndServeTLS(*httpsFlag, *certFlag, *keyFlag, nil)
		}()
	}
	return <-ch
}

var tmpl = template.Must(template.New("").Parse(`
<html>
<head>
<meta name="go-import" content="{{.PkgRoot}} git {{.GitRoot}}">
</head>
<body>
<script>
window.location = "http://godoc.org/{{.PkgPath}}" + window.location.hash;
</script>
</body>
</html>
`))

type Repo struct {
	GitRoot string
	HubRoot string
	PkgRoot string
	PkgPath string
	Version string
}

var re = regexp.MustCompile(`^/([a-z0-9][-a-z0-9]+/)?(v0|v[1-9][0-9]*)/([a-z][-a-z0-9]*)(?:\.git)?((?:/[a-z][-a-z0-9]*)*)$`)

func handler(resp http.ResponseWriter, req *http.Request) {
	log.Printf("%s requested %s", req.RemoteAddr, req.URL)

	if req.URL.Path == "/" {
		resp.Header().Set("Location", "http://godoc.org/gopkg.in/v1/docs")
		resp.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	m := re.FindStringSubmatch(req.URL.Path)
	if m == nil {
		resp.WriteHeader(404)
		return
	}

	var repo *Repo
	if m[1] == "" {
		repo = &Repo{
			GitRoot: "https://github.com/go-" + m[3] + "/" + m[3],
			PkgRoot: "gopkg.in/" + m[2] + "/" + m[3],
			PkgPath: "gopkg.in/" + m[2] + "/" + m[3] + m[4],
			Version: m[2],
		}
	} else {
		repo = &Repo{
			GitRoot: "https://github.com/" + m[1] + m[3],
			PkgRoot: "gopkg.in/" + m[1] + m[2] + "/" + m[3],
			PkgPath: "gopkg.in/" + m[1] + m[2] + "/" + m[3] + m[4],
			Version: m[2],
		}
	}

	repo.HubRoot = repo.GitRoot

	refs, err := hackedRefs(repo)
	switch err {
	case nil:
		repo.GitRoot = "https://" + repo.PkgRoot
	case ErrNoRepo:
		repo.GitRoot += "-" + repo.Version
		repo.HubRoot += "-" + repo.Version
	case ErrNoVersion:
		log.Print(err)
		resp.WriteHeader(http.StatusNotFound)
		return
	default:
		log.Print(err)
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if m[4] == "/git-upload-pack" {
		resp.Header().Set("Location", repo.HubRoot+"/git-upload-pack")
		resp.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if m[4] == "/info/refs" {
		resp.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		resp.Write(refs)
		return
	}

	resp.Header().Set("Content-Type", "text/html")
	tmpl.Execute(resp, repo)
}

var ErrNoRepo = errors.New("repository not found in github")
var ErrNoVersion = errors.New("version reference not found in github")

func hackedRefs(repo *Repo) (data []byte, err error) {
	resp, err := http.Get(repo.HubRoot + ".git/info/refs?service=git-upload-pack")
	if err != nil {
		return nil, fmt.Errorf("cannot talk to github: %v", err)
	}
	switch resp.StatusCode {
	case 200:
		defer resp.Body.Close()
	case 401, 404:
		return nil, ErrNoRepo
	default:
		return nil, fmt.Errorf("error from github: %v", resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading from github: %v", err)
	}

	vhead := "refs/heads/" + repo.Version
	vtag := "refs/tags/" + repo.Version

	var mrefi, mrefj, vrefi, vrefj int

	sdata := string(data)
	for i, j := 0, 0; i < len(data); i = j {
		size, err := strconv.ParseInt(sdata[i:i+4], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse refs line size: %s", string(data[i:i+4]))
		}
		if size == 0 {
			size = 4
		}
		j = i + int(size)
		if j > len(sdata) {
			return nil, fmt.Errorf("incomplete refs data received from github")
		}
		if sdata[0] == '#' {
			continue
		}

		hashi := i + 4
		hashj := strings.IndexByte(sdata[hashi:j], ' ')
		if hashj < 0 || hashj != 40 {
			continue
		}
		hashj += hashi

		namei := hashj + 1
		namej := strings.IndexAny(sdata[namei:j], "\n\x00")
		if namej < 0 {
			namej = j
		} else {
			namej += namei
		}

		name := sdata[namei:namej]

		if name == "refs/heads/master" {
			mrefi = hashi
			mrefj = hashj
		}

		if name == vtag || name == vhead {
			vrefi = hashi
			vrefj = hashj
		}

		if mrefi > 0 && vrefi > 0 {
			break
		}
	}

	if mrefi == 0 || vrefi == 0 {
		return nil, ErrNoVersion
	}

	copy(data[mrefi:mrefj], data[vrefi:vrefj])
	return data, nil
}
