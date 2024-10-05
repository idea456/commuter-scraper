package solver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type SolverServer struct {
	url string
}

type RequestPage struct {
	Cmd        string `json:"cmd"`
	Url        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

type ResponsePage struct {
	Solution ResponseSolutionPage `json:"solution"`
}

type ResponseSolutionPage struct {
	Response string         `json:"response"`
	Cookies  map[string]any `json:"cookies"`
}

func pingSolverServer(url string) error {
	options := RequestPage{
		Cmd:        "request.get",
		Url:        "http://google.com",
		MaxTimeout: 60000,
	}

	optionB, err := json.Marshal(options)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(optionB))
	if err != nil {
		return err
	}

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func NewSolverServer() (*SolverServer, error) {
	url := os.Getenv("SCRAPE_SOLVER_URL")
	if url == "" {
		return nil, fmt.Errorf("You forgot to set SCRAPE_SOLVER_URL")
	}

	err := pingSolverServer(url)
	if err != nil {
		return nil, fmt.Errorf("unable to ping solver server, are you sure the server is still alive?: %w", err)
	}

	return &SolverServer{url: url}, nil
}
