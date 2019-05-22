package swagger

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func FetchRemoteSpec(url string) (*Spec, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not read remote spec: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %v", err)
	}
	return unmarshalSpec(b)
}

func FetchLocalSpec(filename string) (*Spec, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read file from %s: %v", filename, err)
	}
	return unmarshalSpec(b)
}

func unmarshalSpec(b []byte) (*Spec, error) {
	spec := &Spec{}
	if err := json.Unmarshal(b, spec); err != nil {
		return nil, fmt.Errorf("could not parse spec from json: %v", err)
	}
	return spec, nil
}
