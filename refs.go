package gopkg

import(
	"net/http"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
)

const refsSuffix = ".git/info/refs?service=git-upload-pack"

var ErrNoRepo = errors.New("repository not found in GitHub")
var ErrNoVersion = errors.New("version reference not found in GitHub")

func HackedRefs(client *http.Client, repo *Repo) (data []byte, versions []Version, err error) {
	resp, err := client.Get("https://" + repo.GitHubRoot() + refsSuffix)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot talk to GitHub: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		// ok
	case 401, 404:
		return nil, nil, ErrNoRepo
	default:
		return nil, nil, fmt.Errorf("error from GitHub: %v", resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading from GitHub: %v", err)
	}

	var mrefi, mrefj int
	var vrefi, vrefj int
	var vrefv = InvalidVersion

	versions = make([]Version, 0)
	sdata := string(data)
	for i, j := 0, 0; i < len(data); i = j {
		size, err := strconv.ParseInt(sdata[i:i+4], 16, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot parse refs line size: %s", string(data[i:i+4]))
		}
		if size == 0 {
			size = 4
		}
		j = i + int(size)
		if j > len(sdata) {
			return nil, nil, fmt.Errorf("incomplete refs data received from GitHub")
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

		if strings.HasPrefix(name, "refs/heads/v") || strings.HasPrefix(name, "refs/tags/v") {
			if strings.HasSuffix(name, "^{}") {
				// Annotated tag is peeled off and overrides the same version just parsed.
				name = name[:len(name)-3]
			}
			v, ok := ParseVersion(name[strings.IndexByte(name, 'v'):])
			if ok && repo.MajorVersion.Contains(v) && (v == vrefv || !vrefv.IsValid() || vrefv.Less(v)) {
				vrefv = v
				vrefi = hashi
				vrefj = hashj
			}
			if ok {
				versions = append(versions, v)
			}
		}
	}

	// If there were absolutely no versions, and v0 was requested, accept the master as-is.
	if len(versions) == 0 && repo.MajorVersion == (Version{0, -1, -1}) {
		return data, nil, nil
	}

	if mrefi == 0 || vrefi == 0 {
		return nil, nil, ErrNoVersion
	}

	copy(data[mrefi:mrefj], data[vrefi:vrefj])
	return data, versions, nil
}
