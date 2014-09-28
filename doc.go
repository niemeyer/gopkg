// Package gopkg implements stable API's for the Go language.
//
// The service running at:
//
// http://gopkg.in
//
// Is a small command based around this package's functionality. By using this
// package you can integrate versioned import paths into your own domain in
// complex situations (e.g. running a gopkg.in server but on a different domain
// and having a custom home page, etc).
//
// The most basic way to use this package is to create a handler, for instance:
//
//  var pkgHandler = &gopkg.Handler{
//      Host: "gopkg.in",
//      Client: &http.Client{
//          Timeout: 10 * time.Second,
//      },
//  }
//
// Once you have a handler you can ask gopkg to handle any HTTP request:
//
//  // Ask the gopkg handler to handle the HTTP request, if it can.
//  repo, handled := pkgHandler.Handle(resp, req)
//  if handled {
//      // The request was handled by gopkg.
//      return
//  }
//  if repo != nil {
//      // The request was not handled by gopkg, but it is a request for the
//      // package page (e.g. when entering gopkg.in/pkg.v1 in a browser).
//      return
//  }
//
// By default each Handler will use the DefaultMatcher which resolves paths in
// the exact same way that http://gopkg.in does.
//
//  example.com/pkg.v3 → github.com/go-pkg/pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/user/pkg.v3 → github.com/user/pkg   (branch/tag v3, v3.N, or v3.N.M)
//
// If your domain solely represents your GitHub user/organization, you might
// want it to operate differently, for example:
//
//  example.com/pkg.v3 → github.com/exampleorg/pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/folder/pkg.v3 → github.com/exampleorg/folder-pkg   (branch/tag v3, v3.N, or v3.N.M)
//
// This is possible by specifying an single-user URL matcher when creating your
// Handler:
//
//  var pkgHandler = &gopkg.Handler{
//      ...
//      Matcher: gopkg.SingleUser("exampleorg"),
//      ...
//  }
//
package gopkg
