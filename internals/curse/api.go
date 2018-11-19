package curse

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

const baseAPI = "https://staging_cursemeta.dries007.net/api/v3/direct"

var client = &http.Client{}

// FetchMod gets a single mod from metacurse
func FetchMod(id uint32) (*Mod, error) {
	res, _ := ghettoGet(baseAPI + "/addon/" + idToString(id))
	b, _ := ioutil.ReadAll(res.Body)

	mod := Mod{}

	if err := json.Unmarshal(b, &mod); err != nil {
		return nil, err
	}

	return &mod, nil
}

// FetchModFiles gets all files from a single mod
func FetchModFiles(id uint32) ([]ModFile, error) {
	res, _ := ghettoGet(baseAPI + "/addon/" + idToString(id) + "/files")
	b, _ := ioutil.ReadAll(res.Body)

	var modFiles []ModFile

	if err := json.Unmarshal(b, &modFiles); err != nil {
		return nil, err
	}

	return modFiles, nil
}

func idToString(i uint32) string {
	return strconv.Itoa(int(i))
}

// ghettoGet is a helper that does a get request and also sets the User-Agent header
func ghettoGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "minepkg (https://github.com/fiws/minepkg)")
	return client.Do(req)
}
