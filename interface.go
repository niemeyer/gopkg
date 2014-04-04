package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
)

const tmplStrPackage = `<!DOCTYPE html>
<html >
	<head>
		<meta charset="utf-8">
		<title>{{.Repo.Pkg}}.{{.Repo.Version}}{{.Repo.Path}} - {{.Repo.PkgPath}}</title>
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
		<div id="wrap" >
			<div class="container" >
				<div class="row" >
					<div class="col-sm-12" >
						<div class="page-header">
							<h1>{{.Repo.PkgPath}}</h1>
						</div>
					</div>
				</div>
				<div class="row" >
					<div class="col-sm-12" >
						<a class="btn btn-lg btn-info" href="{{.Repo.HubRoot}}/tree/{{if .Repo.Versions}}{{.FullVersion.String}}{{else}}master{{end}}{{.Repo.Path}}" ><i class="fa fa-github"></i> Source Code</a>
						<a class="btn btn-lg btn-info" href="http://godoc.org/{{.Repo.PkgPath}}" ><i class="fa fa-info-circle"></i> API Documentation</a>
					</div>
				</div>
				<div class="row main" >
					<div class="col-sm-8 info" >
						<div class="getting-started" >
							<h2>Getting started</h2>
							<div>
								<p>To get the package, execute:</p>
								<pre>go get {{.Repo.PkgPath}}</pre>
							</div>
							<div>
								<p>To import this package, add the following line to your code:</p>
								<pre>import "{{.Repo.PkgPath}}"</pre>
							</div>
						</div>
					</div>
					<div class="col-sm-3 col-sm-offset-1 versions" >
						<h2>Versions</h2>
						{{ if .LatestVersions }}
							{{ range .LatestVersions }}
								<div>
									<a href='//{{$.Repo.PkgBase}}.v{{.Major}}' {{if eq .Major $.Repo.Version.Major}}class="current"{{end}} >v{{.Major}}</a>
									&rarr;
									<span class="label label-default">{{.String}}</span>
								</div>
							{{ end }}
						{{ else }}
							<div>
								<a href='//{{$.Repo.PkgBase}}.v0' {{if eq 0 $.Repo.Version.Major}}class="current"{{end}} >v0</a>
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

var tmplPackage *template.Template

func init() {
	var err error
	tmplPackage, err = template.New("page").Parse(tmplStrPackage)
	if err != nil {
		fmt.Printf("fatal: parsing template failed: %s\n", err)
		os.Exit(1)
	}
}

type dataPackage struct {
	Repo           *Repo
	LatestVersions VersionList // Contains only the latest version for each major
	FullVersion    Version     // Version that the major requested resolves to
}

func renderInterface(resp http.ResponseWriter, req *http.Request, repo *Repo) {
	data := &dataPackage{
		Repo: repo,
	}
	latestVersionsMap := make(map[int]Version)
	for _, v := range repo.Versions {
		v2, exists := latestVersionsMap[v.Major]
		if !exists || v2.Less(v) {
			latestVersionsMap[v.Major] = v
		}
	}
	data.FullVersion = latestVersionsMap[repo.Version.Major]
	data.LatestVersions = make(VersionList, 0, len(latestVersionsMap))
	for _, v := range latestVersionsMap {
		data.LatestVersions = append(data.LatestVersions, v)
	}
	sort.Sort(sort.Reverse(data.LatestVersions))

	err := tmplPackage.Execute(resp, data)
	if err != nil {
		log.Printf("error executing tmplPackage: %s\n", err)
	}
}
