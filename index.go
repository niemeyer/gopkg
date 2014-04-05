package main

const indexHTML = `<!DOCTYPE html>
<html >
	<head>
		<meta charset="utf-8">
		<title>gopkg.in</title>
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

			.version-example {
				padding-bottom: 5px;
			}

			.resources div {
				padding-top: 5px;
			}
			.resources a {
				width: 100%;
				text-align: left;
			}
			.resources i {
				width: 20px;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="row" >
				<div class="col-sm-12" >
					<div class="page-header">
						<h1>gopkg.in</h1>
					</div>
				</div>
			</div>
			<div class="row">
				<div class="col-sm-9">
					<h1>Introduction</h1>

					<p class="lead" >
						The gopkg.in service provides versioned URLs that offer the proper metadata 
						for redirecting the go tool onto well defined GitHub repositories. Developers 
						that choose to use this service are strongly encouraged to not make any 
						backwards incompatible changes without also changing the version in the 
						package URL. This convention improves the chances that dependent code will 
						continue to work while depended upon packages evolve.
					</p>

					<p>
						The advantage of using gopkg.in is that the URL is cleaner, shorter, redirects
						to the package documentation at godoc.org when opened with a browser, handles
						git branches and tags for versioning, and most importantly encourages the
						adoption of stable versioned package APIs.
					</p>

					<p>
						Note that gopkg.in does not hold the package code. Instead, the go tool is
						redirected and obtains the code straight from the respective GitHub repository.
					</p>

					<h1>Example</h1>

					<p>
						The yaml package may be installed by running:
						<pre>go get gopkg.in/yaml.v1</pre>
					</p>

					<p>
						Although a version selector is provided as part of the import path, code importing
						it must still refer to the Go package as yaml, its actual name.
					</p>

					<p>
						Opening that same URL in a browser will redirect to its documentation at godoc.org:<br/>
						<a href="https://gopkg.in/yaml.v1" >https://gopkg.in/yaml.v1</a>
					</p>

					<p>
						But the actual implementation of the package is in GitHub:<br/>
						<a href="https://github.com/go-yaml/yaml" >https://github.com/go-yaml/yaml</a>
					</p>


					<p>User-owned repositories also work, as described in "Supported URLs" below.</p>

					<p>
						GitHub repositories that have no version tags or branches are considered
						to be unstable, and thus in "v0". See the "Version zero" section for details.
					</p>

					<h1>Supported URLs</h1>

					<p>There are two URL patterns supported:</p>

					<pre>
gopkg.in/pkg.v3 => github.com/go-pkg/pkg (branch/tag "v3", "v3.N", or "v3.N.M")
gopkg.in/user/pkg.v3 => github.com/user/pkg (branch/tag "v3", "v3.N", or "v3.N.M)</pre>

					<p>Path to nested packages may be appended to these URLs.</p>


					<h1>Version number</h1>

					<p>
						The number used in the gopkg.in URL looks like "v1" or "v42", and represents
						the major version for the Go package. No incompatible changes should be done
						to the package without also changing that version, so that packages and
						applications that import the package can continue to work over time without
						being affected by broken dependencies.
					</p>

					<p>
						When using branches or tags to version the GitHub repository, gopkg.in
						understands that a selector in the URL such as "v1" may be satisfied by a
						tag or branch "v1.2" or "v1.2.1" (vMAJOR[.MINOR[.PATCH]]) in the repository,
						and will select the highest version satisfying the requested selector.
					</p>

					<p>
						Even when using richer versioning schemes, the version selector used in
						the gopkg.in URL is restricted to the major version number to ensure that
						all packages in a dependency tree that depend on a given versioned URL will
						pick the same package version whenever possible, rather than importing
						slightly different versions of that package (v1.0.1 and v1.0.2, for example).
						The only situation when such duplicated imports may occur is when the package
						versions are incompatible (v1 and v3, for example), in which case allowing
						the independent imports may provide a chance for the software to still work.
					</p>

					<p>
						For clarity, assuming a repository containing the following tags or branches:
						<div class="version-example"><span class="label label-default">v1</span></div>
						<div class="version-example"><span class="label label-default">v2.0</span></div>
						<div class="version-example"><span class="label label-default">v2.0.3</span></div>
						<div class="version-example"><span class="label label-default">v2.1.2</span></div>
						<div class="version-example"><span class="label label-default">v3</span></div>
						<div class="version-example"><span class="label label-default">v3.0</span></div>
					</p>

					

					<p>
						The following selectors would be resolved as indicated:
						<div class="version-example">pkg.<strong>v1</strong> &rarr; <span class="label label-default">v1</span></div>
						<div class="version-example">pkg.<strong>v2</strong> &rarr; <span class="label label-default">v2.1.2</span></div>
						<div class="version-example">pkg.<strong>v3</strong> &rarr; <span class="label label-default">v3.0</span></div>
					</p>


					<h1>Version zero</h1>

					<p>
						Version zero (v0) is reserved for packages that are so immature that offering
						any kind of API stability guarantees would be unreasonable. This is equivalent
						to labeling the package as alpha or beta quality, and as such the use of these
						packages as dependencies of stable packages and applications is discouraged.
					</p>

					<p>
						Packages should not remain in v0 for too long, as the lack of API stability
						hinders their adoption, and hurts the stability of packages and applications
						that depend on them.
					</p>

					<p>
						Repositories in GitHub that have no version tag or branch matching the
						pattern described above are also considered unstable, and thus gopkg.in takes
						their master branch as v0. This should only be used once the package maintainers
						encourage the use of gopkg.in, though.
					</p>


					<h1>How to change the version</h1>

					<p>
						Increasing the version number is done either by registering a new repository
						in GitHub with the proper name, or by creating a git tag or branch with the
						proper name and pushing it. Which of these are used depends on the choosen
						convention (see the Supported URLs section above).
					</p>

					<p>
						In either case, the previous version should not be removed, so that existent
						code that depends on it remains working. This also preserves the documentation
						for the previous version at godoc.org.
					</p>

					<p>The GitHub documentation details how to push a tag or branch to a remote:</p>

					<a href="https://help.github.com/articles/pushing-to-a-remote" >https://help.github.com/articles/pushing-to-a-remote</a>


					<h3>When to change the version</h3>

					<p>
						The major version should be increased whenever the respective package API
						is being changed in an incompatible way.
					</p>

					<p>Examples of modifications that DO NEED a major version change are:</p>

					<ul>
						<li>Removing or renaming *any* exposed name (function, method, type, etc)</li>
						<li>Adding, removing or renaming methods in an interface</li>
						<li>Adding a parameter to a function, method, or interface</li>
						<li>Changing the type of a parameter or result in a function, method, or interface</li>
						<li>Changing the number of results in a function, method, or interface</li>
						<li>Some struct changes (see details below)</li>
					</ul>

					<p>
						On the other hand, the following modifications are FINE WITHOUT a major version
						change:
					</p>

					<ul>
						<li>Adding exposed names (function, method, type, etc)</li>
						<li>Renaming a parameter or result of a function, method, or interface</li>
						<li>Some struct changes (see details below)</li>
					</ul>

					<p>
						Note that some of these changes may still break code that depends on details
						of the existing API due to the use of reflection. These uses are considered
						an exception, and authors of such packages will have to follow the development
						of dependend upon packages more closely.
					</p>

					<p>
						There are also cases that require further consideration. Some of these are
						covered below.
					</p>


					<h3>Compatibility when changing structs</h3>

					<p>
						Adding a field to a struct may be safe or not depending on how the struct was
						previously defined.
					</p>

					<p>
						A struct containing non-exported fields may always receive new exported fields
						safely, as the language disallows code outside the package from using literals
						of these structs without naming the fields explicitly.
					</p>

					<p>
						A struct with a significant number of fields is likely safe as well, as it's
						inconvenient to be both written and read without naming the fields explicitly.
					</p>

					<p>
						A struct consisting only of a few exported fields must not have fields added
						(exported or not) or repositioned without a major version change, as that
						will break code using such structs in literals with positional fields.
					</p>

					<p>
						Removing or renaming exported fields from an existing struct is of course a
						compatibility breaking change.
					</p>


					<h1>Contact</h1>
					<a href="mailto:gustavo@niemeyer.net">gustavo@niemeyer.net</a> - <a href="https://github.com/niemeyer/gopkg/issues">Github issues</a>
				</div>
				<div class="col-sm-3 resources" >
					<h1>Resources</h1>
					<div>
						<a class="btn btn-info" href="https://github.com/niemeyer/gopkg" ><i class="fa fa-github"></i> Source</a>
					</div>
					<div>
						<a class="btn btn-info" href="https://github.com/niemeyer/gokpg/issues" ><i class="fa fa-question"></i> Issues & Features</a>
					</div>
					<div>
						<a class="btn btn-info" href="http://stats.pingdom.com/r29i3cfl66c0" ><i class="fa fa-signal"></i> Uptime report</a>
					</div>
					<div>
						<a class="btn btn-info" href="mailto:gustavo@niemeyer.net"><i class="fa fa-envelope"></i> gustavo@niemeyer.net</a>
					</div>
				</div>
			</div>
		</div>
	</body>
</html>
`
