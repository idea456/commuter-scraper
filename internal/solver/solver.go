package solver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Solver struct {
	client  *http.Client
	url     string
	session string
}
type SolverRequest struct {
	Cmd        string `json:"cmd"`
	Url        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
	Session    string `json:"session"`
}

type SolverSolutionResponse struct {
	Response string `json:"response"`
}

type SolverResponse struct {
	Solution SolverSolutionResponse `json:"solution"`
}

type SessionResponse struct {
	Session string `json:"session"`
}

func NewSolver(solverUrl string) (*Solver, error) {
	client := &http.Client{
		Timeout: 18000000 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   180000000 * time.Second,
				KeepAlive: 6000000 * time.Second,
				DualStack: true,
			}).DialContext,
		},
	}

	solver := Solver{
		client: client,
		url:    solverUrl,
	}

	session, err := solver.RequestSession()
	if err != nil {
		return nil, err
	}
	solver.session = session

	return &solver, nil
}

func (s *Solver) RequestSession() (string, error) {
	slog.Debug("requesting new session...")
	options := SolverRequest{
		Cmd: "sessions.create",
	}
	optionsB, err := json.Marshal(options)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", s.url, bytes.NewBuffer(optionsB))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error(err.Error())
		return "", err
	}

	var body SessionResponse
	json.NewDecoder(resp.Body).Decode(&body)
	return body.Session, nil
}

// func (s *Solver) RequestPageWithCookie(pageUrl string) (string, error) {

// }

func (s *Solver) RequestPage(pageUrl string) (string, error) {
	slog.Debug(fmt.Sprintf("requesting new page %s...", pageUrl))

	options := SolverRequest{
		Cmd:        "request.get",
		Url:        pageUrl,
		MaxTimeout: 9999999999,
		Session:    s.session,
	}
	optionsB, err := json.Marshal(options)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", s.url, bytes.NewBuffer(optionsB))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}

	var body SolverResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", err
	}

	return body.Solution.Response, nil
}

func (s *Solver) DestroySession() error {
	options := SolverRequest{
		Cmd:     "sessions.destroy",
		Session: s.session,
	}
	optionsB, err := json.Marshal(options)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.url, bytes.NewBuffer(optionsB))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	_, err = s.client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
