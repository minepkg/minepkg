package curse

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

const baseAPI = "https://staging_cursemeta.dries007.net/api/v3/direct"

func idToString(i uint32) string {
	return strconv.Itoa(int(i))
}

// FetchMod gets a single mod from metacurse
func FetchMod(id uint32) (*Mod, error) {
	res, _ := http.Get(baseAPI + "/addon/" + idToString(id))
	b, _ := ioutil.ReadAll(res.Body)

	mod := Mod{}

	if err := json.Unmarshal(b, &mod); err != nil {
		return nil, err
	}

	return &mod, nil
}

// FetchModFiles gets all files from a single mod
func FetchModFiles(id uint32) ([]ModFile, error) {
	res, _ := http.Get(baseAPI + "/addon/" + idToString(id) + "/files")
	b, _ := ioutil.ReadAll(res.Body)

	var modFiles []ModFile

	if err := json.Unmarshal(b, &modFiles); err != nil {
		return nil, err
	}

	return modFiles, nil
}
