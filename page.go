package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
)

const packageTemplateString = `<!DOCTYPE html>
<html >
	<head>
		<meta charset="utf-8">
		<title>{{.Repo.Name}}.{{.Repo.MajorVersion}}{{.Repo.SubPath}} - {{.Repo.GopkgPath}}</title>
		<link href='//fonts.googleapis.com/css?family=Ubuntu+Mono|Ubuntu' rel='stylesheet' >
		<link href="//netdna.bootstrapcdn.com/font-awesome/4.0.3/css/font-awesome.css" rel="stylesheet" >
		<link href="//netdna.bootstrapcdn.com/bootstrap/3.1.1/css/bootstrap.min.css" rel="stylesheet" >
		<style>
			html,
			body {
				height: 100%;
			}

			@media (min-width: 1200px) {
				.container {
					width: 970px;
				}
			}

			body {
				font-family: 'Ubuntu', sans-serif;
			}

			pre {
				font-family: 'Ubuntu Mono', sans-serif;
			}

			.main {
				padding-top: 20px;
			}

			.getting-started div {
				padding-top: 12px;
			}

			.getting-started p {
				font-size: 1.3em;
			}

			.getting-started pre {
				font-size: 15px;
			}

			.versions {
				font-size: 1.3em;
			}
			.versions div {
				padding-top: 5px;
			}
			.versions a {
				font-weight: bold;
			}
			.versions a.current {
				color: black;
				font-decoration: none;
			}

			/* wrapper for page content to push down footer */
			#wrap {
				min-height: 100%;
				height: auto !important;
				height: 100%;
				/* negative indent footer by it's height */
				margin: 0 auto -40px;
			}

			/* footer styling */
			#footer {
				height: 40px;
				background-color: #eee;
				padding-top: 8px;
				text-align: center;
			}

			/* footer fixes for mobile devices */
			@media (max-width: 767px) {
				#footer {
					margin-left: -20px;
					margin-right: -20px;
					padding-left: 20px;
					padding-right: 20px;
				}
			}
		</style>
	</head>
	<body>
		<script type="text/javascript">
			// If there's a URL fragment, assume it's an attempt to read a specific documentation entry. 
			if (window.location.hash.length > 1) {
				window.location = "http://godoc.org/{{.Repo.GopkgPath}}" + window.location.hash;
			}
		</script>
		<div id="wrap" >
			<div class="container" >
				<div class="row" >
					<div class="col-sm-12" >
						<div class="page-header">
							<h1>{{.Repo.GopkgPath}}</h1>
						</div>
					</div>
				</div>
				<div class="row" >
					<div class="col-sm-12" >
						<a class="btn btn-lg btn-info" href="https://{{.Repo.GitHubRoot}}/tree/{{if .Repo.AllVersions}}{{.FullVersion}}{{else}}master{{end}}{{.Repo.SubPath}}" ><i class="fa fa-github"></i> Source Code</a>
						<a class="btn btn-lg btn-info" href="http://godoc.org/{{.Repo.GopkgPath}}" ><i class="fa fa-info-circle"></i> API Documentation</a>
					</div>
				</div>
				<div class="row main" >
					<div class="col-sm-8 info" >
						<div class="getting-started" >
							<h2>Getting started</h2>
							<div>
								<p>To get the package, execute:</p>
								<pre>go get {{.Repo.GopkgPath}}</pre>
							</div>
							<div>
								<p>To import this package, add the following line to your code:</p>
								<pre>import "{{.Repo.GopkgPath}}"</pre>
								{{if .PackageName}}<p>Refer to it as <i>{{.PackageName}}</i>.{{end}}
							</div>
							<div>
								<p>For more details, see the API documentation.</p>
							</div>
						</div>
					</div>
					<div class="col-sm-3 col-sm-offset-1 versions" >
						<h2>Versions</h2>
						{{ if .LatestVersions }}
							{{ range .LatestVersions }}
								<div>
									<a href="//{{gopkgVersionRoot $.Repo .}}{{$.Repo.SubPath}}" {{if eq .Major $.Repo.MajorVersion.Major}}class="current"{{end}} >v{{.Major}}</a>
									&rarr;
									<span class="label label-default">{{.}}</span>
								</div>
							{{ end }}
						{{ else }}
							<div>
								<a href="//{{$.Repo.GopkgPath}}" class="current">v0</a>
								&rarr;
								<span class="label label-default">master</span>
							</div>
						{{ end }}
					</div>
				</div>
			</div>
		</div>

		<div id="footer">
			<div class="container">
				<div class="row">
					<div class="col-sm-12">
						<p class="text-muted credit"><a href="https://gopkg.in">gopkg.in<a></p>
					</div>
				</div>
			</div>
		</div>

		<!--<script src="//ajax.googleapis.com/ajax/libs/jquery/2.1.0/jquery.min.js"></script>-->
		<!--<script src="//netdna.bootstrapcdn.com/bootstrap/3.1.1/js/bootstrap.min.js"></script>-->
	</body>
</html>`

var packageTemplate *template.Template

func gopkgVersionRoot(repo *Repo, version Version) string {
	return repo.GopkgVersionRoot(version)
}

var packageFuncs = template.FuncMap{
	"gopkgVersionRoot": gopkgVersionRoot,
}

func init() {
	var err error
	packageTemplate, err = template.New("page").Funcs(packageFuncs).Parse(packageTemplateString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: parsing package template failed: %s\n", err)
		os.Exit(1)
	}
}

type packageData struct {
	Repo           *Repo
	LatestVersions VersionList // Contains only the latest version for each major
	FullVersion    Version     // Version that the major requested resolves to
	PackageName    string      // Actual package identifier as specified in http://golang.org/ref/spec#PackageClause
}

var regexpPackageName = regexp.MustCompile(`\<h2 id="pkg-overview"\>package ([\p{L}_][\pL\p{Nd}_]*)\</h2\>`)

func renderPackagePage(resp http.ResponseWriter, req *http.Request, repo *Repo) {
	data := &packageData{
		Repo: repo,
	}
	latestVersionsMap := make(map[int]Version)
	for _, v := range repo.AllVersions {
		v2, exists := latestVersionsMap[v.Major]
		if !exists || v2.Less(v) {
			latestVersionsMap[v.Major] = v
		}
	}
	data.FullVersion = latestVersionsMap[repo.MajorVersion.Major]
	data.LatestVersions = make(VersionList, 0, len(latestVersionsMap))
	for _, v := range latestVersionsMap {
		data.LatestVersions = append(data.LatestVersions, v)
	}
	sort.Sort(sort.Reverse(data.LatestVersions))

	godocResp, err := http.Get("http://godoc.org/" + repo.GopkgPath())
	if err == nil {
		godocRespBytes, err := ioutil.ReadAll(godocResp.Body)
		godocResp.Body.Close()
		if err == nil {
			matches := regexpPackageName.FindSubmatch(godocRespBytes)
			if len(matches) == 2 {
				data.PackageName = string(matches[1])
			}
		}
	}

	err = packageTemplate.Execute(resp, data)
	if err != nil {
		log.Printf("error executing tmplPackage: %s\n", err)
	}
}
