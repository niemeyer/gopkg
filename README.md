### Introduction

The gopkg.in service provides versioned URLs that offer the proper metadata for
redirecting the go tool onto well defined GitHub repositories. Developers that
choose to use this service are strongly encouraged to not make any backwards
incompatible changes without also changing the version in the package URL. This
convention improves the chances that dependent code will continue to work while
depended upon packages evolve.

The advantage of using gopkg.in is that the URL is cleaner, shorter, redirects
to the package documentation at godoc.org when opened with a browser, handles
git branches and tags for versioning, and most importantly encourages the
adoption of stable versioned package APIs.

Note that gopkg.in does not hold the package code. Instead, the go tool is
redirected and obtains the code straight from the respective GitHub repository.


### Example

The yaml package may be installed by running:

    go get gopkg.in/yaml.v1

Although a version selector is provided as part of the import path, code
importing it must still refer to the Go package as yaml, its actual name.

Opening that same URL in a browser will redirect to its documentation at
godoc.org:

    https://gopkg.in/yaml.v1

But the actual implementation of the package is in GitHub:

    https://github.com/go-yaml/yaml

User-owned repositories also work, as described in "Supported URLs" below.

GitHub repositories that have no version tags or branches are considered to be
unstable, and thus in "v0". See the "Version zero" section for details.


### Supported URLs

There are two URL patterns supported:

    gopkg.in/pkg.v3      => github.com/go-pkg/pkg (branch/tag "v3", "v3.N", or "v3.N.M")
    gopkg.in/user/pkg.v3 => github.com/user/pkg   (branch/tag "v3", "v3.N", or "v3.N.M)

Path to nested packages may be appended to these URLs.


Version number

The number used in the gopkg.in URL looks like "v1" or "v42", and represents the
major version for the Go package. No incompatible changes should be done to the
package without also changing that version, so that packages and applications
that import the package can continue to work over time without being affected by
broken dependencies.

When using branches or tags to version the GitHub repository, gopkg.in
understands that a selector in the URL such as "v1" may be satisfied by a tag or
branch "v1.2" or "v1.2.1" (vMAJOR[.MINOR[.PATCH]]) in the repository, and will
select the highest version satisfying the requested selector.

Even when using richer versioning schemes, the version selector used in the
gopkg.in URL is restricted to the major version number to ensure that all
packages in a dependency tree that depend on a given versioned URL will pick the
same package version whenever possible, rather than importing slightly different
versions of that package (v1.0.1 and v1.0.2, for example). The only situation
when such duplicated imports may occur is when the package versions are
incompatible (v1 and v3, for example), in which case allowing the independent
imports may provide a chance for the software to still work.

For clarity, assuming a repository containing the following tags or branches:

    v1
    v2.0
    v2.0.3
    v2.1.2
    v3
    v3.0

The following selectors would be resolved as indicated:

    v1  =>  v1
    v2  =>  v2.1.2
    v3  =>  v3.0


Version zero

Version zero (v0) is reserved for packages that are so immature that offering
any kind of API stability guarantees would be unreasonable. This is equivalent
to labeling the package as alpha or beta quality, and as such the use of these
packages as dependencies of stable packages and applications is discouraged.

Packages should not remain in v0 for too long, as the lack of API stability
hinders their adoption, and hurts the stability of packages and applications
that depend on them.

Repositories in GitHub that have no version tag or branch matching the pattern
described above are also considered unstable, and thus gopkg.in takes their
master branch as v0. This should only be used once the package maintainers
encourage the use of gopkg.in, though.


How to change the version

Increasing the version number is done either by registering a new repository in
GitHub with the proper name, or by creating a git tag or branch with the proper
name and pushing it. Which of these are used depends on the choosen convention
(see the Supported URLs section above).

In either case, the previous version should not be removed, so that existent
code that depends on it remains working. This also preserves the documentation
for the previous version at godoc.org.

The GitHub documentation details how to push a tag or branch to a remote:

    https://help.github.com/articles/pushing-to-a-remote


When to change the version

The major version should be increased whenever the respective package API is
being changed in an incompatible way.

Examples of modifications that DO NEED a major version change are:

    * Removing or renaming *any* exposed name (function, method, type, etc)
    * Adding, removing or renaming methods in an interface
    * Adding a parameter to a function, method, or interface
    * Changing the type of a parameter or result in a function, method, or interface
    * Changing the number of results in a function, method, or interface
    * Some struct changes (see details below)

On the other hand, the following modifications are FINE WITHOUT a major version
change:

    * Adding exposed names (function, method, type, etc)
    * Renaming a parameter or result of a function, method, or interface
    * Some struct changes (see details below)

Note that some of these changes may still break code that depends on details of
the existing API due to the use of reflection. These uses are considered an
exception, and authors of such packages will have to follow the development of
dependend upon packages more closely.

There are also cases that require further consideration. Some of these are
covered below.


Compatibility when changing structs

Adding a field to a struct may be safe or not depending on how the struct was
previously defined.

A struct containing non-exported fields may always receive new exported fields
safely, as the language disallows code outside the package from using literals
of these structs without naming the fields explicitly.

A struct with a significant number of fields is likely safe as well, as it's
inconvenient to be both written and read without naming the fields explicitly.

A struct consisting only of a few exported fields must not have fields added
(exported or not) or repositioned without a major version change, as that will
break code using such structs in literals with positional fields.

Removing or renaming exported fields from an existing struct is of course a
compatibility breaking change.


### Contact

gustavo@niemeyer.net

## Usage
